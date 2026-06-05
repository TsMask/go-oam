package socket

import (
	"context"
	"io"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// === UDP 读缓冲池（64KB 覆盖 IP 包上限）===

var udpReadPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 64*1024)
		return &buf
	},
}

// === ClientUDP ===

// ClientUDP UDP 客户端。零值可用，配置项为导出字段，必须在 Connect 之前赋值。
//
// 字段使用约束：
//
//	Addr         必填，服务端地址
//	Port         必填，服务端端口
//	DialTimeout  可选，拨号超时；0 表示 10s（UDP 实际是关联远端，无握手）
//	ReadTimeout  可选，单次 Read 底层超时；0 表示不限
//	WriteTimeout 可选，单次 Write 底层超时；0 表示不限
//	Context      可选，外层 ctx；取消时联动关闭连接
//
// Read 一次返回一个完整 datagram（不等齐、不拼包）；流式拼装由调用方负责。
//
// 生命周期：init → Connect → connected → Close → closed（不可重用）。
//
// 并发约束：
//   - Write 内部串行化（writeMu），并发安全
//   - Read 与 Write 可并发
//   - Close 任何时候可调用，唤醒所有阻塞
//   - 同一 Client 的 Read 不保证 datagram 顺序匹配某个 Write（UDP 协议语义）
type ClientUDP struct {
	// === 配置项（必须在 Connect 之前赋值；运行期修改不安全） ===
	Addr         string         // 必填：服务端地址
	Port         string         // 必填：服务端端口
	DialTimeout  time.Duration  // 拨号超时；0 表示 10s
	ReadTimeout  time.Duration  // 单次 Read 底层超时；0 表示不限
	WriteTimeout time.Duration  // 单次 Write 底层超时；0 表示不限
	Context      context.Context // 可选：外层 ctx；取消时联动关闭连接

	// === 运行时字段 ===
	state     atomic.Int32
	mu        sync.Mutex
	conn      net.Conn
	writeMu   sync.Mutex
	output    chan []byte
	closed    chan struct{}
	onError   func(error)
	closeOnce sync.Once
	readWG    sync.WaitGroup
}

// === 状态查询 ===

// State 返回当前客户端状态（clientState*）。
func (c *ClientUDP) State() int32 { return c.state.Load() }

// IsConnected 返回当前是否已连接。
func (c *ClientUDP) IsConnected() bool { return c.state.Load() == clientStateConnected }

// === 错误监听 ===

// OnError 注册错误监听回调；可在任意时刻调用，多次调用后者覆盖。
// 触发场景同 ClientTCP：readLoop 内的读错误（除用户主动 Close 外）。
func (c *ClientUDP) OnError(fn func(error)) {
	c.mu.Lock()
	c.onError = fn
	c.mu.Unlock()
}

func (c *ClientUDP) dispatchError(err error) {
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

// Connect 关联到 UDP 服务端地址并启动后台读取循环。
//   - 已连接：返回 nil（幂等）
//   - 已关闭：返回 ErrClientClosed（不可重用）
//   - 拨号失败：返回原始 error，state 仍是 init，可重试
func (c *ClientUDP) Connect() error {
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
	conn, err := net.DialTimeout("udp", net.JoinHostPort(c.Addr, c.Port), timeout)
	if err != nil {
		return err
	}

	c.readWG.Add(1)
	c.conn = conn
	c.output = make(chan []byte, 64)
	c.closed = make(chan struct{})
	c.state.Store(clientStateConnected)
	go c.readLoop(conn)
	return nil
}

// shutdown 切 state 到 closed 并清理资源；幂等。语义与 ClientTCP 一致。
func (c *ClientUDP) shutdown(userInitiated bool, readErr error) {
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

// Close 关闭连接并唤醒所有阻塞中的 Read；幂等。
// 注意：output channel 中尚未被 Read 消费的 datagram 会被丢弃，调用方需自行保证 Close 之前已读完必要数据。
func (c *ClientUDP) Close() {
	c.shutdown(true, nil)
	c.readWG.Wait()
}

// contextOrBg 返回 c.Context，未设置时用 background。
func (c *ClientUDP) contextOrBg() context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

// === 后台读取循环 ===

// readLoop 持续从 conn 读取 datagram 推入 output；每次 Read 返回一个完整包。
// 退出路径与 ClientTCP 一致：closed 信号 / Read 错误 / panic recover。
func (c *ClientUDP) readLoop(conn net.Conn) {
	defer c.readWG.Done()
	defer close(c.output)
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 16*1024) // 16KB 覆盖典型 handler 调用栈；runtime.Stack 截断时不报错
			n := runtime.Stack(stack, false)
			c.shutdown(false, fmt.Errorf("readLoop panic: %v\n%s", r, stack[:n]))
		}
	}()

	stop := context.AfterFunc(c.contextOrBg(), func() {
		_ = conn.SetDeadline(time.Now())
	})
	defer stop()

	bufPtr := udpReadPool.Get().(*[]byte)
	defer udpReadPool.Put(bufPtr)
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

// === 原始读写 ===

// Write 发送一个 datagram；writeMu 串行化防止并发字节交错。
func (c *ClientUDP) Write(data []byte) (int, error) {
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

// Read 阻塞读取下一个 datagram。
// closed 状态返 io.EOF（与 ClientTCP 一致：靠 OnError 区分对端断 vs 自己关）。
func (c *ClientUDP) Read() ([]byte, error) {
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

// stateErr UDP 客户端无独立定义，复用 ClientTCP 的 stateErr 通过类型断言不安全；
// 改用内联实现。closed 状态返 ErrClientClosed：与 ClientTCP 语义一致。
func (c *ClientUDP) stateErr() error {
	switch c.state.Load() {
	case clientStateConnected:
		return nil
	case clientStateClosed:
		return ErrClientClosed
	default:
		return ErrClientNotConnected
	}
}