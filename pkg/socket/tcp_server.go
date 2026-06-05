package socket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// === 服务端生命周期状态 ===

const (
	serverStateInit    int32 = iota // 0：未启动
	serverStateServing              // 1：运行中
	serverStateClosed               // 2：完全关闭
)

// === ServerTCP ===

// ServerTCP TCP 服务端。零值可用，配置项为导出字段，必须在 Listen 之前赋值。
//
// 字段使用约束：
//
//	Handler      必填，连接处理函数；返回 error 或 panic 都会通过 OnError 输出
//	OnError      可选：错误监听回调
//	MaxConns     可选：最大并发连接数；0 表示不限
//	TCPKeepAlive 可选：TCP KeepAlive 周期；0 表示系统默认
//	Context      可选：服务端级 ctx；取消时联动关停
//
// 触发 OnError 的场景：handler 返回非 nil error、handler panic（转 error 含 stack）、
// accept 循环的瞬时错误、MaxConns 拒绝连接。
//
// 生命周期：init → Listen → serving → Close → closed。Close 阻塞到所有 handler goroutine 退出。
//
// 并发约束：Handler 内部若需要共享资源须自行同步。
type ServerTCP struct {
	// === 配置项（必须在 Listen 之前赋值；运行期修改不安全） ===
	Handler      func(*Conn) error // 必填：连接处理函数
	OnError      func(error)       // 可选：错误监听回调
	MaxConns     int               // 最大并发连接数；0 表示不限
	TCPKeepAlive time.Duration     // TCP KeepAlive 周期；0 表示系统默认
	Context      context.Context   // 可选：服务端级 ctx；取消时联动关停

	// === 运行时字段 ===
	state        atomic.Int32
	listenerMu   sync.RWMutex
	listener     net.Listener
	connsMu      sync.Mutex
	conns        map[*Conn]struct{}
	sem          chan struct{}
	wg           sync.WaitGroup
	listenWG     sync.WaitGroup
	serverCtx    context.Context
	serverCancel context.CancelFunc
}

// === 状态查询 ===

// State 返回当前服务端状态（serverState*）。
func (s *ServerTCP) State() int32 { return s.state.Load() }

// ConnCount 返回当前活跃连接数。
func (s *ServerTCP) ConnCount() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
}

// ListenAddr 返回实际监听地址；未 Listen 或已 Close 时返回 nil。
func (s *ServerTCP) ListenAddr() net.Addr {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// === 启动 ===

// Listen 监听 address 并服务，阻塞运行直到 Close 或监听出错。
//
// address 格式与 net.Listen 一致：":9090" / "0.0.0.0:9090" / "[::1]:9090"。
//
// 返回值：
//   - nil：Close 触发的正常停止
//   - ErrServerClosed：已 Close 后再次调用
//   - ErrAlreadyServing：并发启动
//   - ErrNoHandler：未设置 Handler
//   - 其他：net.Listen 失败或 accept 不可恢复错误
//
// 关停路径：Close 关 listener → Accept 返回 ErrClosed → defer 等所有 handler 退出。
func (s *ServerTCP) Listen(address string) error {
	if s.Handler == nil {
		return ErrNoHandler
	}
	if !s.state.CompareAndSwap(serverStateInit, serverStateServing) {
		cur := s.state.Load()
		if cur == serverStateClosed {
			return ErrServerClosed
		}
		return ErrAlreadyServing
	}

	s.listenWG.Add(1)

	// 先 net.Listen：失败时不分配 ctx，避免 context.WithCancel + AfterFunc 资源 leak
	ln, err := net.Listen("tcp", address)
	if err != nil {
		s.state.Store(serverStateClosed)
		return err
	}

	// 初始化运行时字段（仅在 listen 成功后）
	parent := s.Context
	if parent == nil {
		parent = context.Background()
	}
	s.listenerMu.Lock()
	s.serverCtx, s.serverCancel = context.WithCancel(parent)
	s.listener = ln
	s.listenerMu.Unlock()
	s.connsMu.Lock()
	s.conns = make(map[*Conn]struct{})
	s.connsMu.Unlock()
	if s.MaxConns > 0 {
		s.sem = make(chan struct{}, s.MaxConns)
	}

	// ctx 取消时关闭 listener，Accept 立即返回 ErrClosed
	stopWatch := context.AfterFunc(s.serverCtx, s.closeListener)
	defer stopWatch()

	defer s.listenWG.Done()
	defer func() {
		s.closeListener()
		s.serverCancel()
		s.wg.Wait() // 此时不会再有 wg.Add，安全
		s.state.Store(serverStateClosed)
	}()

	for {
		raw, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			// 瞬时错误：通知 OnError + 短退避重试；关停信号到达时立即退出
			s.dispatchError(fmt.Errorf("accept: %w", err))
			select {
			case <-s.serverCtx.Done():
				return nil
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		if s.sem != nil {
			select {
			case s.sem <- struct{}{}:
			default:
				s.rejectConn(raw)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConn(raw)
	}
}

// rejectConn 在并发已满时发送提示再关闭，避免客户端只看到 RST。
func (s *ServerTCP) rejectConn(raw net.Conn) {
	_ = raw.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = raw.Write([]byte("ERROR: server connection limit reached\r\n"))
	_ = raw.Close()
	s.dispatchError(fmt.Errorf("connection rejected: max conns %d reached", s.MaxConns))
}

// === 连接处理 ===

// handleConn 单连接主流程。handler 返回 error 或 panic 都通过 dispatchError 输出。
func (s *ServerTCP) handleConn(raw net.Conn) {
	defer s.wg.Done()

	if s.TCPKeepAlive > 0 {
		if tc, ok := raw.(*net.TCPConn); ok {
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(s.TCPKeepAlive)
		}
	}

	s.listenerMu.RLock()
	parent := s.serverCtx
	s.listenerMu.RUnlock()
	connCtx, cancel := context.WithCancel(parent)
	// ctx 取消时立即设 deadline 唤醒阻塞中的 Read/Write
	stop := context.AfterFunc(connCtx, func() {
		_ = raw.SetDeadline(time.Now())
	})

	conn := &Conn{
		server: s,
		raw:    raw,
		ctx:    connCtx,
		cancel: cancel,
	}

	s.connsMu.Lock()
	s.conns[conn] = struct{}{}
	s.connsMu.Unlock()
	defer func() {
		s.connsMu.Lock()
		delete(s.conns, conn)
		s.connsMu.Unlock()
		_ = conn.Close()
		stop()
	}()

	var handlerErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := make([]byte, 16*1024) // 16KB 覆盖典型 handler 调用栈；runtime.Stack 截断时不报错
				n := runtime.Stack(stack, false)
				handlerErr = fmt.Errorf("handler panic from %s: %v\n%s", conn.RemoteAddr(), r, stack[:n])
			}
		}()
		handlerErr = s.Handler(conn)
	}()
	if handlerErr != nil {
		s.dispatchError(handlerErr)
	}
}

// dispatchError 调用 OnError 回调；未设置 OnError 时直接返回。回调内 panic 被吞掉。
func (s *ServerTCP) dispatchError(err error) {
	if s.OnError == nil {
		return
	}
	defer func() { _ = recover() }()
	s.OnError(err)
}

// === 关停 ===

// Close 关闭服务端，阻塞直到所有 handler goroutine 退出。幂等。
//
// 流程：CAS 切到 closed → 关 listener（让 Accept 立即返回 ErrClosed）
//
//	→ 取消 serverCtx（ctx.AfterFunc 唤醒所有 handler 的阻塞 IO）
//	→ 强关所有活跃 conn
//	→ listenWG.Wait 等 Listen goroutine 完全退出（Listen defer 内会 wg.Wait 等所有 handler）。
//
// 未启动过 Listen 时仅切状态后返回（无资源需释放）。
func (s *ServerTCP) Close() {
	if s.state.CompareAndSwap(serverStateInit, serverStateClosed) {
		return
	}
	if !s.state.CompareAndSwap(serverStateServing, serverStateClosed) {
		return
	}

	s.closeListener()
	s.listenerMu.Lock()
	if s.serverCancel != nil {
		s.serverCancel()
	}
	s.listenerMu.Unlock()
	s.closeAllConns()
	s.listenWG.Wait()
}

// closeListener 关闭并清空 listener，幂等。
func (s *ServerTCP) closeListener() {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
}

// closeAllConns 强关所有活跃连接。先 snapshot 再解锁，避免长持锁。
func (s *ServerTCP) closeAllConns() {
	s.connsMu.Lock()
	snapshot := make([]*Conn, 0, len(s.conns))
	for c := range s.conns {
		snapshot = append(snapshot, c)
	}
	s.connsMu.Unlock()
	for _, c := range snapshot {
		_ = c.Close()
	}
}

// === Conn ===

// Conn 服务端侧的连接封装。
//
// 与 net.Conn 的区别：
//   - Context() 与服务端生命周期联动，关停时自动取消并设 deadline 唤醒阻塞 IO
//   - Close() 幂等且与 Server 注册表联动
//   - Server() 拿到所属 ServerTCP 实例
//
// 并发约束（重要）：
//   - Read 与 Read 可并发（底层 net.TCPConn 支持）
//   - Write 与 Write 不可并发：net.TCPConn.Write 不保证并发安全，并发调用会损坏字节流
//   - Read 与 Write 可并发
// 业务层若需多 goroutine 写同一连接，必须自行加锁。
type Conn struct {
	server *ServerTCP
	raw    net.Conn
	ctx    context.Context
	cancel context.CancelFunc
	closed atomic.Bool
}

// Read 读取数据；连接已关闭返回 net.ErrClosed；ctx 取消时返回 ctx.Err。
func (c *Conn) Read(b []byte) (int, error) {
	if c.closed.Load() {
		return 0, net.ErrClosed
	}
	n, err := c.raw.Read(b)
	if err != nil && c.ctx.Err() != nil {
		return n, c.ctx.Err()
	}
	return n, err
}

// Write 写入数据；连接已关闭返回 net.ErrClosed；ctx 取消时返回 ctx.Err。
func (c *Conn) Write(b []byte) (int, error) {
	if c.closed.Load() {
		return 0, net.ErrClosed
	}
	n, err := c.raw.Write(b)
	if err != nil && c.ctx.Err() != nil {
		return n, c.ctx.Err()
	}
	return n, err
}

// Close 关闭连接，幂等。先 cancel ctx 再关底层连接。
func (c *Conn) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.cancel()
	return c.raw.Close()
}

// Context 返回连接级 ctx，关停时自动取消。
func (c *Conn) Context() context.Context { return c.ctx }

// Server 返回所属 ServerTCP 实例。
func (c *Conn) Server() *ServerTCP { return c.server }

// RemoteAddr 返回客户端地址。
func (c *Conn) RemoteAddr() net.Addr { return c.raw.RemoteAddr() }

// LocalAddr 返回本地地址。
func (c *Conn) LocalAddr() net.Addr { return c.raw.LocalAddr() }

// IsClosed 返回当前连接是否已关闭。
func (c *Conn) IsClosed() bool { return c.closed.Load() }
