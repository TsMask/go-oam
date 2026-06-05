// Package telnet 提供 telnet 客户端与服务端实现。
//
// 服务端示例（字段配置风格，配置项必须在 Listen 之前赋值）：
//
//	s := &telnet.Server{
//	    Handler:      echoHandler,
//	    OnError:      func(err error) { log.Println(err) },
//	    MaxConns:     100,
//	    TCPKeepAlive: 30 * time.Second,
//	}
//	go func() { _ = s.Listen(":2323") }()
//	defer s.Close()
//
// 关键约定：
//   - 配置项（Handler/OnError/MaxConns/TCPKeepAlive）必须在 Listen 之前赋值
//   - handler 返回 error 或 panic 统一通过 OnError 输出（panic 自动 recover 并携带 stack）
//   - Close 阻塞到所有 handler goroutine 退出，期间通过 ctx 取消唤醒阻塞 IO
package telnet

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

// === 公共错误 ===

var (
	// ErrServerClosed 服务端已关闭（或正在关闭）。
	ErrServerClosed = errors.New("telnet: server closed")
	// ErrAlreadyServing 服务端已经在服务中，禁止重复启动。
	ErrAlreadyServing = errors.New("telnet: server already serving")
	// ErrNoHandler Listen 前未设置 Handler 字段。
	ErrNoHandler = errors.New("telnet: handler not set")
)

// === 服务端生命周期状态 ===

const (
	serverStateInit    int32 = iota // 0：未启动
	serverStateServing              // 1：运行中
	serverStateClosed               // 2：完全关闭
)

// === Server ===

// Server telnet 服务端。零值可用，所有配置项为导出字段，必须在 Listen 之前赋值。
//
// 字段使用约束：
//
//	Handler      必填，连接处理函数；返回 error 或 panic 都会通过 OnError 输出
//	OnError      可选，错误回调；回调内 panic 被吞掉避免影响其他连接
//	MaxConns     可选，最大并发连接数；0 表示不限
//	TCPKeepAlive 可选，TCP KeepAlive 周期；0 表示使用系统默认
//
// 触发 OnError 的场景：handler 返回非 nil error、handler panic（转 error 含 stack）、
// accept 循环的瞬时错误、MaxConns 拒绝连接。
type Server struct {
	// === 配置项（必须在 Listen 之前赋值；运行期修改不安全） ===
	Handler      func(*Conn) error // 必填：连接处理函数
	OnError      func(error)       // 可选：错误监听回调
	MaxConns     int               // 最大并发连接数；0 表示不限
	TCPKeepAlive time.Duration     // TCP KeepAlive 周期；0 表示使用系统默认

	// === 运行时字段（私有，由 Listen/Close 维护） ===
	state        atomic.Int32       // 服务端状态机（serverState*）
	listenerMu   sync.RWMutex       // 保护 listener / serverCtx / serverCancel
	listener     net.Listener       // 监听器；Close 后置 nil
	connsMu      sync.Mutex         // 保护 conns map
	conns        map[*Conn]struct{} // 活跃连接集合
	sem          chan struct{}      // MaxConns 信号量；nil 表示不限
	wg           sync.WaitGroup     // 跟踪 handleConn goroutine；仅 Listen defer 内调用 Wait
	listenWG     sync.WaitGroup     // 跟踪 Listen goroutine 自身；Close 用它串行 wg 调用避免语义 race
	serverCtx    context.Context    // 服务端生命周期 ctx
	serverCancel context.CancelFunc // 取消 serverCtx
}

// === 状态查询 ===

// State 返回当前服务端状态（serverState*）。
func (s *Server) State() int32 { return s.state.Load() }

// ConnCount 返回当前活跃连接数。
func (s *Server) ConnCount() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
}

// ListenAddr 返回实际监听地址；未 Listen 或已 Close 时返回 nil。
func (s *Server) ListenAddr() net.Addr {
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
// address 格式与 net.Listen 一致：":2323" / "0.0.0.0:2323" / "[::1]:2323"。
//
// 返回值：
//   - nil：Close 触发的正常停止
//   - ErrServerClosed：已 Close 后再次调用
//   - ErrAlreadyServing：并发启动
//   - ErrNoHandler：未设置 Handler 字段
//   - 其他：net.Listen 失败或 accept 不可恢复错误
//
// 关停路径：Close 关 listener → Accept 返回 ErrClosed → 进入 defer 等所有 handler 退出。
func (s *Server) Listen(address string) error {
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

	// 初始化运行时字段（首次 Listen 时建立）
	s.listenerMu.Lock()
	s.serverCtx, s.serverCancel = context.WithCancel(context.Background())
	s.listenerMu.Unlock()
	s.connsMu.Lock()
	s.conns = make(map[*Conn]struct{})
	s.connsMu.Unlock()
	if s.MaxConns > 0 {
		s.sem = make(chan struct{}, s.MaxConns)
	}

	ln, err := net.Listen("tcp", address)
	if err != nil {
		s.state.Store(serverStateClosed)
		return err
	}

	s.listenerMu.Lock()
	s.listener = ln
	s.listenerMu.Unlock()

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
func (s *Server) rejectConn(raw net.Conn) {
	_ = raw.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = raw.Write([]byte("ERROR: server connection limit reached\r\n"))
	_ = raw.Close()
	s.dispatchError(fmt.Errorf("connection rejected: max conns %d reached", s.MaxConns))
}

// === 连接处理 ===

// handleConn 单连接主流程。handler 返回 error 或 panic 都通过 OnError 输出；
// panic 转 error 时携带 stack；handleConn 退出前从 conns 注册表注销并关闭底层连接。
func (s *Server) handleConn(raw net.Conn) {
	defer s.wg.Done()

	if s.TCPKeepAlive > 0 {
		if tc, ok := raw.(*net.TCPConn); ok {
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(s.TCPKeepAlive)
		}
	}

	// 创建连接级 ctx：父级是 serverCtx，服务端关停时自动联动取消
	s.listenerMu.RLock()
	parent := s.serverCtx
	s.listenerMu.RUnlock()
	connCtx, cancel := context.WithCancel(parent)
	// ctx 取消时立即设置 deadline 唤醒阻塞中的 Read/Write
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

	// 调用 handler：捕获返回 error 与 panic，统一通过 dispatchError 输出
	var handlerErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := make([]byte, 4096)
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

// dispatchError 调用 OnError 回调；未设置 OnError 时直接返回。
// 回调内 panic 被吞掉避免影响其他连接。
func (s *Server) dispatchError(err error) {
	if s.OnError == nil {
		return
	}
	defer func() { _ = recover() }()
	s.OnError(err)
}

// === 关停 ===

// Close 关闭服务端，阻塞直到所有 handler goroutine 退出。
//
// 流程：CAS 切到 closed → 关 listener（让 Accept 立即返回 ErrClosed）
//
//	→ 取消 serverCtx（ctx.AfterFunc 唤醒所有 handler 的阻塞 IO）
//	→ 强关所有活跃 conn
//	→ listenWG.Wait 等 Listen goroutine 完全退出（Listen defer 内会 wg.Wait 等所有 handler）。
//
// 幂等：重复调用直接返回。未启动过 Listen 时仅切状态后返回（无资源需释放）。
//
// 说明：Close 自身不调 wg.Wait，是为了避免与 Listen accept loop 内的 wg.Add(1) 构成
// "Add 在 counter==0 时与 Wait 并发" 的语义 race（Go 文档明确禁止）。
func (s *Server) Close() {
	if s.state.CompareAndSwap(serverStateInit, serverStateClosed) {
		return // 未启动过 Listen，无资源需释放
	}
	if !s.state.CompareAndSwap(serverStateServing, serverStateClosed) {
		return // 已 closed，幂等返回
	}

	s.closeListener()
	// serverCancel 持锁读，与 Listen 内的初始化同步
	s.listenerMu.Lock()
	if s.serverCancel != nil {
		s.serverCancel()
	}
	s.listenerMu.Unlock()
	s.closeAllConns()
	s.listenWG.Wait() // 等 Listen goroutine 退出（其 defer 内 wg.Wait 才安全）
}

// closeListener 关闭并清空 listener，幂等。
func (s *Server) closeListener() {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
}

// closeAllConns 强关所有活跃连接。先 snapshot 再解锁，避免长持锁。
func (s *Server) closeAllConns() {
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
//   - Server() 拿到所属 Server 实例
//
// 并发安全：Read/Write 代理到底层 net.Conn，业务层串行化由调用方负责。
type Conn struct {
	server *Server            // 所属服务端
	raw    net.Conn           // 底层连接
	ctx    context.Context    // 连接级 ctx
	cancel context.CancelFunc // 取消连接级 ctx
	closed atomic.Bool        // 关闭标记，保证 Close 幂等
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

// Server 返回所属 Server 实例。
func (c *Conn) Server() *Server { return c.server }

// RemoteAddr 返回客户端地址。
func (c *Conn) RemoteAddr() net.Addr { return c.raw.RemoteAddr() }

// LocalAddr 返回本地地址。
func (c *Conn) LocalAddr() net.Addr { return c.raw.LocalAddr() }

// IsClosed 返回当前连接是否已关闭。
func (c *Conn) IsClosed() bool { return c.closed.Load() }
