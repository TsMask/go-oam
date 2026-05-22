package telnet

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// 读缓冲区对象池，减少高并发下 GC 压力
var bufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 4096)
		return &buf
	},
}

// Client telnet 客户端，支持并发安全的命令发送
//
// 使用示例:
//
//	c := &telnet.Client{Addr: "192.168.1.1", Port: "23"}
//	if err := c.Connect(); err != nil { panic(err) }
//	defer c.Close()
//
//	// 登录认证（可选）
//	if err := c.Auth("admin", "123"); err != nil { panic(err) }
//
//	// 匹配提示符
//	out, err := c.Send("display version\r\n", 30*time.Second, func(b []byte) bool {
//	    return bytes.HasSuffix(b, []byte(">")) || bytes.HasSuffix(b, []byte("#"))
//	})
type Client struct {
	Addr        string        `json:"addr"` // telnet 地址
	Port        string        `json:"port"` // telnet 端口
	DialTimeOut time.Duration // 连接超时，默认 10s
	MaxRead     int           // 单次 Send 最大读取字节数，默认 1MB

	conn   net.Conn
	connMu sync.RWMutex // 保护 conn 字段
	sendMu sync.Mutex   // 串行化 Send 操作
}

// Connect 连接到 telnet 服务端，重复调用已连接时直接返回 nil
func (c *Client) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil
	}

	// IPv6 地址协议
	proto := "tcp"
	if strings.Contains(c.Addr, ":") {
		proto = "tcp6"
	}

	timeout := c.DialTimeOut
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	conn, err := net.DialTimeout(proto, net.JoinHostPort(c.Addr, c.Port), timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Close 关闭连接，可安全地在 Send 过程中调用（会中断阻塞中的 Read）
func (c *Client) Close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// IsConnected 返回当前连接状态
func (c *Client) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}

// Auth 执行 telnet 登录认证，先读取服务端提示符再发送凭据
//
// user: 用户名，为空则跳过；password: 密码，为空则跳过
func (c *Client) Auth(user, password string) error {
	if user == "" && password == "" {
		return nil
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("telnet client not connected")
	}

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	// 等待登录提示（如 "login:"），超时也继续
	if user != "" {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.Read(*bufPtr) // 忽略错误，超时或无数据均可
		if _, err := conn.Write([]byte(user + "\r\n")); err != nil {
			return fmt.Errorf("auth user write: %w", err)
		}
	}

	// 等待密码提示（如 "Password:"），超时也继续
	if password != "" {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.Read(*bufPtr) // 忽略错误，超时或无数据均可
		if _, err := conn.Write([]byte(password + "\r\n")); err != nil {
			return fmt.Errorf("auth password write: %w", err)
		}
	}

	return nil
}

// WindowChange 通知远端终端窗口大小变更（h 行 x w 列）
func (c *Client) WindowChange(h, w int) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("telnet client not connected")
	}
	// NAWS 协议: IAC WILL NAWS + IAC SB NAWS <wH><wL><hH><hL> IAC SE
	conn.Write([]byte{255, 251, 31})
	conn.Write([]byte{255, 250, 31, byte(w >> 8), byte(w & 0xFF), byte(h >> 8), byte(h & 0xFF), 255, 240})
	return nil
}

// Write 向连接写入原始数据（不做任何包装），并发安全
func (c *Client) Write(cmd []byte) (int, error) {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return 0, fmt.Errorf("telnet client not connected")
	}
	return conn.Write(cmd)
}

// Read 从连接读取一次原始数据（单次读取，不做循环），超时返回 error
//
// 适用于需要精细控制读取节奏的场景，如交互式逐行读取；一般场景请使用 Send
func (c *Client) Read(timeout time.Duration) ([]byte, error) {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("telnet client not connected")
	}

	if timeout == 0 {
		timeout = 1 * time.Second
	}

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	conn.SetReadDeadline(time.Now().Add(timeout))
	n, err := conn.Read(*bufPtr)
	if err != nil {
		return nil, err
	}
	// 拷贝后归还池，避免返回的 slice 引用池内存
	out := make([]byte, n)
	copy(out, (*bufPtr)[:n])
	return out, nil
}

// Send 发送命令并读取响应（并发安全，多个 goroutine 调用自动串行化）
//   - cmd: 要发送的命令
//   - timeout: 读写超时
//   - done: 每次 Read 后调用，传入已累计数据，返回 true 表示读取完成；传 nil 则读到超时/EOF
//
// 使用示例:
//
//	// 匹配提示符
//	out, err := c.Send("display version\r\n", 30*time.Second, func(b []byte) bool {
//	    return bytes.HasSuffix(b, []byte(">")) || bytes.HasSuffix(b, []byte("#"))
//	})
//
//	// 仅超时
//	out, err := c.Send("display version\r\n", 5*time.Second, nil)
func (c *Client) Send(cmd string, timeout time.Duration, done func([]byte) bool) (string, error) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return "", fmt.Errorf("telnet client not connected")
	}

	if timeout == 0 {
		timeout = 1 * time.Second
	}

	maxRead := c.MaxRead
	if maxRead <= 0 {
		maxRead = 1 << 20 // 1MB
	}

	// 写入命令（带超时）
	if cmd != "" {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		if _, err := conn.Write([]byte(cmd)); err != nil {
			return "", err
		}
	}

	// 读取响应（超时设一次，控制总读取时间窗口）
	conn.SetReadDeadline(time.Now().Add(timeout))

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
	tmp := *bufPtr

	result := make([]byte, 0, 4096)
	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			result = append(result, tmp[:n]...)
			if len(result) >= maxRead {
				return string(result[:maxRead]), nil
			}
			if done != nil && done(result) {
				return string(result), nil
			}
		}
		if err != nil {
			if len(result) > 0 {
				return string(result), nil
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return "", fmt.Errorf("timeout")
			}
			return "", err
		}
	}
}
