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

// === ServerUDP ===

// ServerUDP UDP 服务端。零值可用，配置项为导出字段，必须在 Listen 之前赋值。
//
// 字段使用约束：
//
//	Handler  必填：每个到达的 datagram 调用一次；返回 error 或 panic 都通过 OnError 输出
//	OnError  可选：错误监听回调
//	MaxConns 可选：最大并发 handler 数量（UDP 是无连接的，这里限的是 goroutine 并发）
//	Context  可选：服务端级 ctx；取消时联动关停
//
// 触发 OnError 的场景：handler 返回非 nil error、handler panic（转 error 含 stack）、
// ReadFromUDP 错误、达到 MaxConns 丢包。
//
// 生命周期：init → Listen → serving → Close → closed。Close 阻塞到所有 handler goroutine 退出。
//
// 并发约束：handler 可并发执行；同地址的并发回包可能乱序（UDP 协议语义）。
type ServerUDP struct {
	// === 配置项（必须在 Listen 之前赋值；运行期修改不安全） ===
	Handler  func(*PacketConn, []byte, *net.UDPAddr) error // 必填
	OnError  func(error)                                   // 可选
	MaxConns int                                           // 最大并发 handler 数；0 表示不限
	Context  context.Context                               // 可选：服务端级 ctx

	// === 运行时字段 ===
	state         atomic.Int32
	listenerMu    sync.RWMutex
	conn          *net.UDPConn
	sem           chan struct{}
	activeHandler atomic.Int64 // 当前活跃 handler 计数，ConnCount 使用
	wg            sync.WaitGroup
	listenWG      sync.WaitGroup
	serverCtx     context.Context
	serverCancel  context.CancelFunc
}

// === 状态查询 ===

// State 返回当前服务端状态（serverState*）。
func (s *ServerUDP) State() int32 { return s.state.Load() }

// ConnCount 返回当前活跃 handler 数（无论是否限流）。
func (s *ServerUDP) ConnCount() int {
	return int(s.activeHandler.Load())
}

// ListenAddr 返回实际监听地址；未 Listen 或已 Close 时返回 nil。
func (s *ServerUDP) ListenAddr() net.Addr {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	if s.conn == nil {
		return nil
	}
	return s.conn.LocalAddr()
}

// === 启动 ===

// Listen 监听 UDP address 并分发每个 datagram 给 handler，阻塞运行直到 Close。
//
// address 格式与 net.ListenUDP 一致：":9091" / "0.0.0.0:9091"。
//
// 返回值与 ServerTCP.Listen 一致。
func (s *ServerUDP) Listen(address string) error {
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

	// 先 ResolveUDPAddr + ListenUDP：失败时不分配 ctx，避免 context.WithCancel + AfterFunc 资源 leak
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		s.state.Store(serverStateClosed)
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
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
	s.conn = conn
	s.listenerMu.Unlock()
	if s.MaxConns > 0 {
		s.sem = make(chan struct{}, s.MaxConns)
	}

	defer s.listenWG.Done()
	defer func() {
		s.closeConn()
		s.serverCancel()
		s.wg.Wait()
		s.state.Store(serverStateClosed)
	}()

	// 周期性检查 serverCtx 是否取消，避免 ReadFromUDP 永久阻塞
	stopWatch := context.AfterFunc(s.serverCtx, func() {
		_ = conn.SetReadDeadline(time.Now())
	})
	defer stopWatch()

	bufPtr := udpReadPool.Get().(*[]byte)
	defer udpReadPool.Put(bufPtr)
	buf := *bufPtr

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || s.serverCtx.Err() != nil {
				return nil
			}
			s.dispatchError(fmt.Errorf("udp read: %w", err))
			// 关停信号到达时立即退出；其他错误短暂退避
			select {
			case <-s.serverCtx.Done():
				return nil
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}
		if n == 0 {
			continue
		}

		// 并发 handler 数量限制
		if s.sem != nil {
			select {
			case s.sem <- struct{}{}:
			default:
				s.dispatchError(fmt.Errorf("udp max handlers reached: %d, dropping packet", s.MaxConns))
				continue
			}
		}

		// 拷贝 datagram 后再异步分发，避免读缓冲被覆盖
		data := make([]byte, n)
		copy(data, buf[:n])

		s.listenerMu.RLock()
		parentCtx := s.serverCtx
		s.listenerMu.RUnlock()
		connCtx, cancel := context.WithCancel(parentCtx)

		pc := &PacketConn{
			server: s,
			raw:    conn,
			ctx:    connCtx,
			cancel: cancel,
		}

		s.wg.Add(1)
		go s.handlePacket(pc, data, remoteAddr)
	}
}

// handlePacket 单包处理。handler 返回 error 或 panic 都通过 dispatchError 输出。
func (s *ServerUDP) handlePacket(pc *PacketConn, data []byte, addr *net.UDPAddr) {
	defer s.wg.Done()
	defer pc.Close()
	defer func() {
		if s.sem != nil {
			<-s.sem
		}
	}()
	s.activeHandler.Add(1)
	defer s.activeHandler.Add(-1)

	var handlerErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := make([]byte, 16*1024) // 16KB 覆盖典型 handler 调用栈；runtime.Stack 截断时不报错
				n := runtime.Stack(stack, false)
				handlerErr = fmt.Errorf("handler panic from %s: %v\n%s", addr, r, stack[:n])
			}
		}()
		handlerErr = s.Handler(pc, data, addr)
	}()
	if handlerErr != nil {
		s.dispatchError(handlerErr)
	}
}

// dispatchError 调用 OnError 回调；未设置时直接返回。回调内 panic 被吞掉。
func (s *ServerUDP) dispatchError(err error) {
	if s.OnError == nil {
		return
	}
	defer func() { _ = recover() }()
	s.OnError(err)
}

// === 关停 ===

// Close 关闭服务端，阻塞直到所有 handler goroutine 退出。幂等。
func (s *ServerUDP) Close() {
	if s.state.CompareAndSwap(serverStateInit, serverStateClosed) {
		return
	}
	if !s.state.CompareAndSwap(serverStateServing, serverStateClosed) {
		return
	}

	s.closeConn()
	s.listenerMu.Lock()
	if s.serverCancel != nil {
		s.serverCancel()
	}
	s.listenerMu.Unlock()
	s.listenWG.Wait()
}

// closeConn 关闭并清空 UDPConn，幂等。
func (s *ServerUDP) closeConn() {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

// === PacketConn ===

// PacketConn UDP 服务端单包处理上下文。
//
// 与 *net.UDPConn 的区别：
//   - 提供 WriteToUDP 回包方法，自动应用 ctx 取消语义
//   - Context() 与服务端生命周期联动
//   - Server() 拿到所属 ServerUDP 实例
//
// 生命周期：handler 退出时自动 Close；同一 PacketConn 不可跨 handler 共享。
type PacketConn struct {
	server *ServerUDP
	raw    *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
	closed atomic.Bool
}

// WriteToUDP 向 addr 发送 datagram；连接已关闭返回 net.ErrClosed；ctx 取消时返回 ctx.Err。
func (p *PacketConn) WriteToUDP(data []byte, addr *net.UDPAddr) (int, error) {
	if p.closed.Load() {
		return 0, net.ErrClosed
	}
	n, err := p.raw.WriteToUDP(data, addr)
	if err != nil && p.ctx.Err() != nil {
		return n, p.ctx.Err()
	}
	return n, err
}

// Close 关闭 PacketConn，幂等。释放 ctx 资源。
func (p *PacketConn) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.cancel()
	return nil
}

// Context 返回连接级 ctx，关停时自动取消。
func (p *PacketConn) Context() context.Context { return p.ctx }

// Server 返回所属 ServerUDP 实例。
func (p *PacketConn) Server() *ServerUDP { return p.server }

// LocalAddr 返回本地监听地址。
func (p *PacketConn) LocalAddr() net.Addr { return p.raw.LocalAddr() }
