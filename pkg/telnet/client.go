package telnet

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// === Telnet 协议常量 ===

const (
	iacByte byte = 0xFF
	cmdWILL byte = 0xFB
	cmdWONT byte = 0xFC
	cmdDO   byte = 0xFD
	cmdDONT byte = 0xFE
	cmdSB   byte = 0xFA
	cmdSE   byte = 0xF0
	optNAWS byte = 0x1F

	// maxIACPending 限制跨 chunk 边界的 IAC 序列缓冲长度。
	// 防止恶意/异常服务端发送不完整 IAC 导致内存耗尽。
	maxIACPending = 4096
)

// === Client 错误 ===

var (
	// ErrClientClosed Client 已关闭。Client 不可重用。
	ErrClientClosed = errors.New("telnet: client closed")
	// ErrClientNotConnected Client 未建立连接。
	ErrClientNotConnected = errors.New("telnet: not connected")
	// ErrClientTruncated Exec 响应超过 MaxRead 上限被截断。
	ErrClientTruncated = errors.New("telnet: response truncated to MaxRead")
)

// === 读缓冲池 ===

var bufPool = sync.Pool{
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

// === Client ===

// Client telnet 客户端。零值可用，所有配置项为导出字段，必须在 Connect 之前赋值。
//
// 字段使用约束：
//
//	Addr            必填，telnet 服务端地址
//	Port            必填，telnet 服务端端口
//	DialTimeout     可选，拨号超时；0 表示 10s 默认
//	ReadTimeout     可选，单次 Read 底层超时；0 表示不限（依赖 TCP KeepAlive 保护）
//	WriteTimeout    可选，单次 Write 底层超时；0 表示不限
//	TCPKeepAlive    可选，TCP KeepAlive 周期；0 表示使用系统默认
//	MaxRead         可选，Exec 单次最大读取字节数；0 表示 1MB
//	AuthPromptWait  可选，Auth 等待 prompt 的超时；0 表示 2s
//	Newline         可选，Auth 凭据行尾序列；空串表示 "\r\n"
//	KeepIAC         可选，Exec 是否保留原始 IAC 协商字节（默认过滤）
//	OnError(fn)     可选，错误监听回调，通过方法注册；回调内 panic 被吞掉避免影响主流程
//
// 触发 OnError 的场景：readLoop 内 conn.Read 返回非 nil error 且非用户主动 Close
// （远端断开 / 读写错误 / readLoop panic 转 error 携带 stack）。用户主动 Close 不触发。
//
// 生命周期：init → Connect → connected → Close → closed（不可重用）。
// dial 失败时 state 仍是 init，用户可重试 Connect。
//
// 并发约束（仅靠文档，无编程强制）：
//   - Exec / Auth 互不并发（字节流会交错）
//   - Read 与 Exec 互不并发（output channel 单消费者）
//   - Close 任何时候可调用，唤醒所有阻塞
type Client struct {
	// === 配置项（必须在 Connect 之前赋值；运行期修改不安全） ===
	Addr           string        // 必填：telnet 服务端地址
	Port           string        // 必填：telnet 服务端端口
	DialTimeout    time.Duration // 拨号超时；0 表示 10s
	ReadTimeout    time.Duration // 单次 Read 底层超时；0 表示不限
	WriteTimeout   time.Duration // 单次 Write 底层超时；0 表示不限
	TCPKeepAlive   time.Duration // TCP KeepAlive 周期；0 表示系统默认
	MaxRead        int           // Exec 单次最大读取字节数；0 表示 1MB
	AuthPromptWait time.Duration // Auth 等待 prompt 超时；0 表示 2s
	Newline        string        // Auth 凭据行尾序列；空串表示 "\r\n"
	KeepIAC        bool          // Exec 是否保留原始 IAC 协商字节（默认过滤）

	// === 运行时字段（私有，由 Connect/Close/readLoop 维护） ===
	state     atomic.Int32   // 状态机（clientState*）
	onError   func(error)    // 错误回调，通过 OnError 方法设置；mu 保护读写
	mu        sync.Mutex     // 保护 conn / output / closed 字段的初始化与读取
	conn      net.Conn       // 底层连接
	writeMu   sync.Mutex     // 串行化 Write
	output    chan []byte    // readLoop → Read 的数据通道
	closed    chan struct{}  // 关闭信号；shutdown 时 close，唤醒所有阻塞
	closeOnce sync.Once      // 保证 conn.Close + close(closed) 只执行一次
	readWG    sync.WaitGroup // 跟踪 readLoop goroutine；Close 用它串行调用避免 wg 语义 race
}

// === 状态查询 ===

// State 返回当前客户端状态（clientState*）。
func (c *Client) State() int32 { return c.state.Load() }

// IsConnected 返回当前是否已连接。
func (c *Client) IsConnected() bool { return c.state.Load() == clientStateConnected }

// stateErr 根据 state 返回对应 error。
func (c *Client) stateErr() error {
	switch c.state.Load() {
	case clientStateConnected:
		return nil
	case clientStateClosed:
		return ErrClientClosed
	default:
		return ErrClientNotConnected
	}
}

// getConn 持锁读取 conn；连接未建立或已关闭返回 error。
func (c *Client) getConn() (net.Conn, error) {
	if err := c.stateErr(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil, ErrClientNotConnected
	}
	return conn, nil
}

// readChans 持锁读取 output/closed；连接未建立或已关闭返回 error。
func (c *Client) readChans() (<-chan []byte, <-chan struct{}, error) {
	if err := c.stateErr(); err != nil {
		return nil, nil, err
	}
	c.mu.Lock()
	output, closed := c.output, c.closed
	c.mu.Unlock()
	if output == nil || closed == nil {
		return nil, nil, ErrClientNotConnected
	}
	return output, closed, nil
}

// === 错误监听 ===

// OnError 注册错误监听回调；可在任意时刻调用，多次调用后者覆盖。
//
// 触发场景：
//   - readLoop 内 conn.Read 返回非 nil error 且非用户主动 Close（远端断开/读写错误）
//   - readLoop panic（转 error 携带 stack）
//
// 用户主动调用 Close 不会触发 OnError。回调内 panic 被吞掉避免影响主流程。
func (c *Client) OnError(fn func(error)) {
	c.mu.Lock()
	c.onError = fn
	c.mu.Unlock()
}

// dispatchError 派发错误给 OnError；未设置 OnError 时直接返回。
// 回调内 panic 被吞掉避免影响主流程。
func (c *Client) dispatchError(err error) {
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
//   - 已关闭：返回 ErrClientClosed（Client 不可重用）
//   - 拨号失败：返回原始 error，state 仍是 init，可重试
func (c *Client) Connect() error {
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
//   - userInitiated=true：用户主动调 Close，不触发 OnError
//   - readErr 非 nil：readLoop 异常退出，触发 OnError
//
// state CAS 保证只有第一个调用方进入清理；closeOnce 保证 conn.Close + close(closed)
// 只执行一次（Close 与 readLoop 异常竞争时仍安全）。
func (c *Client) shutdown(userInitiated bool, readErr error) {
	if !c.state.CompareAndSwap(clientStateConnected, clientStateClosed) {
		return // 已被另一方抢先关闭
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
		c.dispatchError(fmt.Errorf("read: %w", readErr))
	}
}

// Close 关闭连接并唤醒所有阻塞中的 Read；幂等。
// Client 关闭后不可再用，如需重连请创建新 Client。用户主动调用本方法不会触发 OnError。
//
// 内部先 shutdown（切 state + 关 conn + close closed）再 readWG.Wait 等 readLoop 退出，
// 与 Server 的 Close/listenWG 模型一致：避免 wg 语义 race。
func (c *Client) Close() {
	c.shutdown(true, nil)
	c.readWG.Wait()
}

// === 后台读取循环 ===

// readLoop 持续从 conn 读入 output channel；退出时关闭 output。
//
// 退出路径：
//   - c.closed 关闭（用户主动 Close 触发 shutdown）：state 已切，shutdown CAS 失败，直接 return
//   - conn.Read 返回 error 且 state 还是 connected：调 shutdown 触发 OnError
//   - panic：recover 转 error 携带 stack 调 shutdown
func (c *Client) readLoop(conn net.Conn) {
	defer c.readWG.Done()
	defer close(c.output)
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			n := runtime.Stack(stack, false)
			c.shutdown(false, fmt.Errorf("readLoop panic: %v\n%s", r, stack[:n]))
		}
	}()

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
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
			// state 是 closed 表示用户主动 Close；否则是远端断开 / 读写错误
			if c.state.Load() != clientStateClosed {
				c.shutdown(false, err)
			}
			return
		}
	}
}

// === 原始读写 ===

// Write 向连接写入原始数据；writeMu 串行化防止并发字节交错。
// WriteTimeout > 0 时设置写 deadline，防止 hung conn。
func (c *Client) Write(cmd []byte) (int, error) {
	conn, err := c.getConn()
	if err != nil {
		return 0, err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.WriteTimeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	} else {
		_ = conn.SetWriteDeadline(time.Time{}) // 清除 deadline
	}
	return conn.Write(cmd)
}

// Read 阻塞读取下一个数据块。
//   - 有数据：返回 (data, nil)
//   - 连接关闭且 output 已 drain：返回 (nil, io.EOF)
//   - 调用方超时：通过 select + time.After 实现（无内置超时）
//
// closed 信号到达时会再 drain 一次 output，避免 close 与写入并发丢数据。
func (c *Client) Read() ([]byte, error) {
	output, closed, err := c.readChans()
	if err != nil {
		return nil, err
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

// === 命令发送 ===

// Exec 发送命令并读取响应直到 done 命中或连接关闭。
//   - cmdStr 空字符串表示只读不写
//   - done nil 则读到连接关闭
//   - 响应超过 MaxRead 返回 ErrClientTruncated
//   - 调用方超时：通过 select 控制
func (c *Client) Exec(cmdStr string, done func([]byte) bool) (string, error) {
	if err := c.stateErr(); err != nil {
		return "", err
	}
	if cmdStr != "" {
		if _, err := c.Write([]byte(cmdStr)); err != nil {
			return "", fmt.Errorf("write cmd: %w", err)
		}
	}

	maxRead := c.MaxRead
	if maxRead <= 0 {
		maxRead = 1 << 20
	}

	var iac iacProcessor
	if c.KeepIAC {
		iac.disable()
	}

	result := make([]byte, 0, 4096)
	for {
		data, err := c.Read()
		if err != nil {
			if len(result) > 0 {
				return string(result), nil
			}
			return "", err
		}
		result = append(result, iac.feed(data)...)
		if len(result) >= maxRead {
			return string(result[:maxRead]), ErrClientTruncated
		}
		if done != nil && done(result) {
			return string(result), nil
		}
	}
}

// === 认证 ===

// Auth 执行 telnet 登录认证。
//   - user/password 任一为空则跳过对应步骤；都为空直接返回 nil
//   - 每次发送凭据前会等待 AuthPromptWait 读取 prompt，超时则继续发送
//
// 注意：drainPrompt 只消费一条 chunk；prompt 跨 chunk 时可能错位。
func (c *Client) Auth(user, password string) error {
	if user == "" && password == "" {
		return nil
	}
	if err := c.stateErr(); err != nil {
		return err
	}

	newline := c.Newline
	if newline == "" {
		newline = "\r\n"
	}
	wait := c.AuthPromptWait
	if wait == 0 {
		wait = 2 * time.Second
	}

	if user != "" {
		if err := c.drainPrompt(wait); err != nil {
			return err
		}
		if _, err := c.Write([]byte(user + newline)); err != nil {
			return fmt.Errorf("auth user write: %w", err)
		}
	}
	if password != "" {
		if err := c.drainPrompt(wait); err != nil {
			return err
		}
		if _, err := c.Write([]byte(password + newline)); err != nil {
			return fmt.Errorf("auth password write: %w", err)
		}
	}
	return nil
}

// drainPrompt 等待并丢弃一条 prompt 数据；超时返回 nil。
func (c *Client) drainPrompt(timeout time.Duration) error {
	output, closed, err := c.readChans()
	if err != nil {
		return err
	}
	timer := time.NewTimer(timeout)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()
	select {
	case _, ok := <-output:
		if !ok {
			return io.EOF
		}
		return nil
	case <-timer.C:
		return nil
	case <-closed:
		return io.EOF
	}
}

// === 终端控制 ===

// WindowChange 通知远端终端窗口大小变更（h 行 x w 列）。
func (c *Client) WindowChange(h, w int) error {
	if _, err := c.Write(nawsMessage(h, w)); err != nil {
		return fmt.Errorf("window change write: %w", err)
	}
	return nil
}

// nawsMessage 构造 NAWS 协商消息（合并为单包避免 TCP 拆包错位）。
func nawsMessage(h, w int) []byte {
	return []byte{
		iacByte, cmdWILL, optNAWS,
		iacByte, cmdSB, optNAWS,
		byte(w >> 8), byte(w & 0xFF),
		byte(h >> 8), byte(h & 0xFF),
		iacByte, cmdSE,
	}
}

// === IAC 协商字节过滤器 ===

// iacProcessor 状态化的 Telnet IAC 协商字节过滤器。
// 处理跨 chunk 边界的 IAC 序列，超阈值时 flush 防止内存耗尽。
//
// 支持的 IAC 序列：
//   - 2 字节命令: IAC <cmd>
//   - 3 字节命令: IAC WILL/WONT/DO/DONT <opt>
//   - 子协商:    IAC SB ... IAC SE
//   - 转义:      IAC IAC 视为字面量 0xFF
type iacProcessor struct {
	disabled bool
	pending  []byte
}

func (p *iacProcessor) disable() {
	p.disabled = true
}

// feed 输入 chunk 返回过滤后的字节。
// 快速路径：pending 为空且 chunk 中不含 0xFF 时直接返回原 chunk，零分配。
func (p *iacProcessor) feed(chunk []byte) []byte {
	if p.disabled {
		return chunk
	}
	if len(p.pending) == 0 && bytes.IndexByte(chunk, iacByte) < 0 {
		return chunk
	}

	data := chunk
	if len(p.pending) > 0 {
		combined := make([]byte, 0, len(p.pending)+len(chunk))
		combined = append(combined, p.pending...)
		combined = append(combined, chunk...)
		p.pending = p.pending[:0]
		data = combined
	}

	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if data[i] != iacByte {
			out = append(out, data[i])
			i++
			continue
		}
		// IAC 起始
		if i+1 >= len(data) {
			return p.stash(data[i:], out)
		}
		switch cmd := data[i+1]; cmd {
		case iacByte:
			out = append(out, iacByte)
			i += 2
		case cmdWILL, cmdWONT, cmdDO, cmdDONT:
			if i+2 >= len(data) {
				return p.stash(data[i:], out)
			}
			i += 3
		case cmdSB:
			j, ok := findSubnegEnd(data, i+2)
			if !ok {
				return p.stash(data[i:], out)
			}
			i = j
		default:
			// 2 字节命令（IAC NOP / IAC BREAK 等）
			i += 2
		}
	}
	return out
}

// stash 把 data 加入 pending；若超过上限则把 pending flush 到 out 并清空。
// 返回最终的 out（调用方应 return 此值）。
func (p *iacProcessor) stash(data, out []byte) []byte {
	p.pending = append(p.pending, data...)
	if len(p.pending) >= maxIACPending {
		out = append(out, p.pending...)
		p.pending = p.pending[:0]
	}
	return out
}

// findSubnegEnd 在 data 中从 start 起查找 IAC SE，返回 SE 之后的索引。
// 找不到完整 SE 时返回 (0, false)。
func findSubnegEnd(data []byte, start int) (int, bool) {
	for j := start; j+1 < len(data); j++ {
		if data[j] == iacByte && data[j+1] == cmdSE {
			return j + 2, true
		}
	}
	return 0, false
}
