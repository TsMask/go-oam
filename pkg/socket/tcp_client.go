// Package socket 提供 TCP/UDP 传输层原语。
//
// 设计原则：
//   - 传输层只暴露 Read/Write/Close，协议层的"读直到 X"或"读 N 字节"由调用方组装
//   - 结构体字面量配置，零值可用，0 表示默认值
//   - Server.OnError 在结构体字面量里赋值，ClientTCP.OnError 通过方法注册
//   - 状态机 init → connected/started → closed，重复启动 / 关闭幂等
//   - 关闭路径走 context，cancel 时自动设 deadline 唤醒阻塞 IO
//
// TCP 用法示例（echo 协议）：
//
//	srv := &socket.ServerTCP{
//	    Handler: func(c *socket.Conn) error {
//	        buf := make([]byte, 4096)
//	        n, err := c.Read(buf)
//	        if err != nil { return err }
//	        _, err = c.Write(buf[:n])
//	        return err
//	    },
//	    OnError: func(err error) { log.Println(err) },
//	    MaxConns: 100,
//	}
//	go func() { _ = srv.Listen(":9090") }()
//	defer srv.Close()
//
//	cli := &socket.ClientTCP{Addr: "127.0.0.1", Port: "9090"}
//	if err := cli.Connect(); err != nil { log.Fatal(err) }
//	defer cli.Close()
//	_, _ = cli.Write([]byte("ping"))
//	data, err := cli.Read()
//
// UDP 用法示例（echo 协议）：
//
//	srv := &socket.ServerUDP{
//	    Handler: func(pc *socket.PacketConn, data []byte, addr *net.UDPAddr) error {
//	        _, err := pc.WriteToUDP(data, addr)
//	        return err
//	    },
//	}
//	go func() { _ = srv.Listen(":9091") }()
//	defer srv.Close()
//
//	cli := &socket.ClientUDP{Addr: "127.0.0.1", Port: "9091"}
//	if err := cli.Connect(); err != nil { log.Fatal(err) }
//	defer cli.Close()
//	_, _ = cli.Write([]byte("ping"))
//	data, err := cli.Read()
package socket

import (
	"context"
	"io"
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
	// ErrClientClosed ClientTCP 已关闭。ClientTCP 不可重用。
	ErrClientClosed = errors.New("socket: ClientTCP closed")
	// ErrClientNotConnected ClientTCP 未建立连接。
	ErrClientNotConnected = errors.New("socket: not connected")
	// ErrServerClosed 服务端已关闭（或正在关闭）。
	ErrServerClosed = errors.New("socket: server closed")
	// ErrServerNotStarted 服务端未 Listen。
	ErrServerNotStarted = errors.New("socket: server not started")
	// ErrAlreadyServing 服务端已经在服务中。
	ErrAlreadyServing = errors.New("socket: already serving")
	// ErrNoHandler Listen 前未设置 Handler。
	ErrNoHandler = errors.New("socket: handler not set")
)

// === 读缓冲池（TCP 16KB；UDP 用 64KB 见 udp_client.go）===

var tcpReadPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 16*1024)
		return &buf
	},
}

// === 客户端生命周期状态 ===

const (
	clientStateInit      int32 = iota // 0：未启动
	clientStateConnected              // 1：已连接
	clientStateClosed                 // 2：终态，不可重用
)

// === ClientTCP ===

// ClientTCP TCP 客户端。零值可用，配置项为导出字段，必须在 Connect 之前赋值。
//
// 字段使用约束：
//
//	Addr         必填，服务端地址
//	Port         必填，服务端端口
//	DialTimeout  可选，拨号超时；0 表示 10s
//	ReadTimeout  可选，单次 Read 底层超时；0 表示不限
//	WriteTimeout 可选，单次 Write 底层超时；0 表示不限
//	TCPKeepAlive 可选，TCP KeepAlive 周期；0 表示系统默认
//	Context      可选，外层 ctx；取消时联动关闭连接
//
// 生命周期：init → Connect → connected → Close → closed（不可重用）。
// dial 失败时 state 仍是 init，可重试 Connect。
//
// 并发约束：
//   - Write 内部串行化（writeMu），并发安全
//   - Read 与 Write 可并发
//   - Close 任何时候可调用，唤醒所有阻塞
//   - OnError 回调内 panic 被吞掉避免影响主流程
type ClientTCP struct {
	// === 配置项（必须在 Connect 之前赋值；运行期修改不安全） ===
	Addr         string         // 必填：服务端地址
	Port         string         // 必填：服务端端口
	DialTimeout  time.Duration  // 拨号超时；0 表示 10s
	ReadTimeout  time.Duration  // 单次 Read 底层超时；0 表示不限
	WriteTimeout time.Duration  // 单次 Write 底层超时；0 表示不限
	TCPKeepAlive time.Duration  // TCP KeepAlive 周期；0 表示系统默认
	Context      context.Context // 可选：外层 ctx；取消时联动关闭连接

	// === 运行时字段（私有，由 Connect/Close/readLoop 维护） ===
	state     atomic.Int32  // 状态机（clientState*）
	mu        sync.Mutex    // 保护 conn / output / closed / onError 的读写
	conn      net.Conn      // 底层连接
	writeMu   sync.Mutex    // 串行化 Write
	output    chan []byte   // readLoop → Read 的数据通道（缓冲 64 块）
	closed    chan struct{} // 关闭信号；shutdown 时 close，唤醒所有阻塞
	onError   func(error)   // 错误回调，通过 OnError 方法设置
	closeOnce sync.Once     // 保证 conn.Close + close(closed) 只执行一次
	readWG    sync.WaitGroup // 跟踪 readLoop goroutine
}

// === 状态查询 ===

// State 返回当前客户端状态（clientState*）。
func (c *ClientTCP) State() int32 { return c.state.Load() }

// IsConnected 返回当前是否已连接。
func (c *ClientTCP) IsConnected() bool { return c.state.Load() == clientStateConnected }

// stateErr 根据 state 返回对应 error。
//
// closed 状态返 ErrClientClosed，与 telnet Client 保持一致：
//   - 区分"用户主动 Close"vs"对端断开"通过 OnError 回调，不通过 Read 返回值
//   - OnError 在对端断开时触发，用户主动 Close 不会触发
//   - Read 在 closed 状态统一返 ErrClientClosed，避免竞态
func (c *ClientTCP) stateErr() error {
	switch c.state.Load() {
	case clientStateConnected:
		return nil
	case clientStateClosed:
		return ErrClientClosed
	default:
		return ErrClientNotConnected
	}
}

// === 错误监听 ===

// OnError 注册错误监听回调；可在任意时刻调用，多次调用后者覆盖。
//
// 触发场景：readLoop 内 conn.Read 返回非 nil error 且非用户主动 Close
// （远端断开 / 读写错误 / readLoop panic 转 error 携带 stack）。
// 用户主动 Close 不会触发。
func (c *ClientTCP) OnError(fn func(error)) {
	c.mu.Lock()
	c.onError = fn
	c.mu.Unlock()
}

// dispatchError 派发错误给 OnError；未设置时直接返回。回调内 panic 被吞掉。
func (c *ClientTCP) dispatchError(err error) {
	c.mu.Lock()
	fn := c.onError
	c.mu.Unlock()
	if fn == nil {
		return
	}
	defer func() { _ = recover() }()
	fn(err)
}

// === 连接生命周期 ===

// Connect 拨号并启动后台读取循环。
//   - 已连接：返回 nil（幂等）
//   - 已关闭：返回 ErrClientClosed（ClientTCP 不可重用）
//   - 拨号失败：返回原始 error，state 仍是 init，可重试
func (c *ClientTCP) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state.Load() {
	case clientStateConnected:
		return nil
	case clientStateClosed:
		return ErrClientClosed
	}

	timeout := c.DialTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(c.Addr, c.Port), timeout)
	if err != nil {
		return err
	}
	if c.TCPKeepAlive > 0 {
		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(c.TCPKeepAlive)
		}
	}

	// readWG.Add 必须在 state 切到 connected 之前，保证 Close 的 Wait 不会先于 Add
	c.readWG.Add(1)
	c.conn = conn
	c.output = make(chan []byte, 64)
	c.closed = make(chan struct{})
	c.state.Store(clientStateConnected)
	go c.readLoop(conn)
	return nil
}

// shutdown 切 state 到 closed 并清理资源；幂等。
//   - userInitiated=true：用户主动 Close，不触发 OnError
//   - readErr 非 nil：readLoop 异常退出，触发 OnError
//
// state CAS 保证只有第一个调用方进入清理；closeOnce 保证 conn.Close + close(closed) 只执行一次。
func (c *ClientTCP) shutdown(userInitiated bool, readErr error) {
	if !c.state.CompareAndSwap(clientStateConnected, clientStateClosed) {
		return
	}
	c.closeOnce.Do(func() {
		c.mu.Lock()
		conn := c.conn
		c.conn = nil
		closed := c.closed
		c.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		if closed != nil {
			close(closed)
		}
	})
	if !userInitiated && readErr != nil {
		c.dispatchError(readErr)
	}
}

// Close 关闭连接并唤醒所有阻塞中的 Read；幂等。ClientTCP 不可重用，如需重连请创建新 ClientTCP。
// 用户主动调用本方法不会触发 OnError。
//
// 注意：output channel 中尚未被 Read 消费的字节会被丢弃，调用方需自行保证 Close 之前已读完必要数据。
func (c *ClientTCP) Close() {
	c.shutdown(true, nil)
	c.readWG.Wait()
}

// === 后台读取循环 ===

// readLoop 持续从 conn 读入 output channel；退出时关闭 output。
//
// 退出路径：
//   - c.closed 关闭（用户主动 Close 触发 shutdown）
//   - conn.Read 返回 error 且 state 还是 connected：调 shutdown 触发 OnError
//   - panic：recover 转 error 携带 stack 调 shutdown
func (c *ClientTCP) readLoop(conn net.Conn) {
	defer c.readWG.Done()
	defer close(c.output)
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 16*1024) // 16KB 覆盖典型 handler 调用栈；runtime.Stack 截断时不报错
			n := runtime.Stack(stack, false)
			c.shutdown(false, fmt.Errorf("readLoop panic: %v\n%s", r, stack[:n]))
		}
	}()

	// ctx 取消时立即设 deadline 唤醒阻塞中的 Read
	stop := context.AfterFunc(c.contextOrBg(), func() {
		_ = conn.SetDeadline(time.Now())
	})
	defer stop()

	bufPtr := tcpReadPool.Get().(*[]byte)
	defer tcpReadPool.Put(bufPtr)
	buf := *bufPtr

	for {
		if c.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
		}
		n, err := conn.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case c.output <- data:
			case <-c.closed:
				return
			}
		}
		if err != nil {
			if c.state.Load() != clientStateClosed {
				// ctx 取消触发的 SetDeadline 错误归为"用户主动"，不触发 OnError
				if c.contextOrBg().Err() != nil {
					c.shutdown(true, err)
				} else {
					c.shutdown(false, err)
				}
			}
			return
		}
	}
}

// contextOrBg 返回 c.Context，未设置时用 background。
func (c *ClientTCP) contextOrBg() context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

// === 原始读写 ===

// Write 向连接写入原始数据；writeMu 串行化防止并发字节交错。
// WriteTimeout > 0 时设置写 deadline，防止 hung conn。
func (c *ClientTCP) Write(data []byte) (int, error) {
	if err := c.stateErr(); err != nil {
		return 0, err
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return 0, ErrClientNotConnected
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.WriteTimeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	} else {
		_ = conn.SetWriteDeadline(time.Time{})
	}
	return conn.Write(data)
}

// Read 阻塞读取下一个数据块。
//   - 有数据：返回 (data, nil)
//   - 连接已结束（用户主动 Close / 对端断开）：返回 (nil, io.EOF)
//   - 从未 Connect：返回 ErrClientNotConnected
//   - 调用方超时：通过外层 select + time.After 实现（无内置超时）
//
// 与 Write 的不同：Write 在 closed 状态返 ErrClientClosed（已死连接明确报错）；
// Read 在 closed 状态返 io.EOF（"对端关 / 自己关"都走同一条 EOF 路径，
// 区分通过 OnError 回调：用户主动 Close 不触发，对端断开触发）。
//
// 这与 telnet Client 行为一致：TestRead_EOFOnClose 验证 Close 后 Read 返 io.EOF。
func (c *ClientTCP) Read() ([]byte, error) {
	// init 状态（从未 Connect）直接报 not connected
	if c.state.Load() == clientStateInit {
		return nil, ErrClientNotConnected
	}
	c.mu.Lock()
	output, closed := c.output, c.closed
	c.mu.Unlock()
	if output == nil || closed == nil {
		return nil, ErrClientNotConnected
	}
	select {
	case data, ok := <-output:
		if !ok {
			return nil, io.EOF
		}
		return data, nil
	case <-closed:
		// closed 信号到达；output 内可能还有未消费数据，drain 一次
		select {
		case data, ok := <-output:
			if !ok {
				return nil, io.EOF
			}
			return data, nil
		default:
			return nil, io.EOF
		}
	}
}


