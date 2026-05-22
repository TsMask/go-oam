package socket

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

// ClientTCP TCP 客户端（Send 操作互斥，支持并发安全）
type ClientTCP struct {
	Addr        string        // 主机地址
	Port        string        // 端口
	DialTimeOut time.Duration // 连接超时，默认 5s
	MaxRead     int           // 单次 Send 最大读取字节数，默认 1MB

	conn   net.Conn
	connMu sync.RWMutex // 保护 conn 字段
	sendMu sync.Mutex   // 串行化 Send 操作
}

// Connect 连接到 TCP 服务端，重复调用已连接时直接返回 nil
func (c *ClientTCP) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil
	}

	proto := "tcp"
	if strings.Contains(c.Addr, ":") {
		proto = "tcp6"
	}

	timeout := c.DialTimeOut
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	conn, err := net.DialTimeout(proto, net.JoinHostPort(c.Addr, c.Port), timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Close 关闭连接，可安全地在 Send 过程中调用（会中断阻塞中的 Read）
func (c *ClientTCP) Close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// IsConnected 返回当前连接状态
func (c *ClientTCP) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}

// Send 发送消息并读取响应（并发安全，多个 goroutine 调用自动串行化）
//   - timeout: 读写超时
//   - done: 每次 Read 后调用，返回 true 表示读取完成；传 nil 则读到超时/EOF
//
// 使用示例:
//
//	// SSH/telnet：匹配提示符
//	data, err := c.Send(cmd, 30*time.Second, func(b []byte) bool {
//	    return bytes.HasSuffix(b, []byte(">")) || bytes.HasSuffix(b, []byte("#"))
//	})
//
//	// 仅超时
//	data, err := c.Send(cmd, 5*time.Second)
func (c *ClientTCP) Send(msg []byte, timeout time.Duration, done func([]byte) bool) ([]byte, error) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("tcp client not connected")
	}

	// 写入（带超时）
	if len(msg) > 0 {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		if _, err := conn.Write(msg); err != nil {
			return nil, err
		}
	}

	// 读取响应
	conn.SetReadDeadline(time.Now().Add(timeout))

	maxRead := c.MaxRead
	if maxRead <= 0 {
		maxRead = 1 << 20 // 1MB
	}

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
	tmp := *bufPtr

	result := make([]byte, 0, 4096)
	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			result = append(result, tmp[:n]...)
			if len(result) >= maxRead {
				return result[:maxRead], nil
			}
			if done != nil && done(result) {
				return result, nil
			}
		}
		if err != nil {
			if len(result) > 0 {
				return result, nil
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, fmt.Errorf("timeout")
			}
			return nil, err
		}
	}
}
