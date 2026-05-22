package socket

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ClientUDP UDP 客户端（Send 操作互斥，支持并发安全）
type ClientUDP struct {
	Addr        string        // 主机地址
	Port        string        // 端口
	DialTimeOut time.Duration // 连接超时，默认 5s
	MaxRead     int           // 单次读取最大字节数，默认 64KB

	conn   net.Conn
	connMu sync.RWMutex
	sendMu sync.Mutex
}

// Connect 连接到 UDP 服务端
func (c *ClientUDP) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil
	}

	proto := "udp"
	if strings.Contains(c.Addr, ":") {
		proto = "udp6"
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

// Close 关闭连接
func (c *ClientUDP) Close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// IsConnected 返回当前连接状态
func (c *ClientUDP) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}

// Send 发送数据报并读取响应（并发安全）
//   - timeout: 读写超时
//   - done: 每次 Read 后调用，返回 true 表示读取完成
//     传 nil 时读取单条数据报即返回；传入函数时循环读取直到 done 返回 true
func (c *ClientUDP) Send(msg []byte, timeout time.Duration, done func([]byte) bool) ([]byte, error) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("udp client not connected")
	}

	// 写入
	if len(msg) > 0 {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		if _, err := conn.Write(msg); err != nil {
			return nil, err
		}
	}

	conn.SetReadDeadline(time.Now().Add(timeout))

	maxRead := c.MaxRead
	if maxRead <= 0 {
		maxRead = 64 << 10 // 64KB
	}

	// done 为 nil：单条数据报即返回
	if done == nil {
		buf := make([]byte, maxRead)
		n, err := conn.Read(buf)
		if n > 0 {
			// 拷贝到精确大小 slice，避免底层数组泄漏
			result := make([]byte, n)
			copy(result, buf[:n])
			return result, nil
		}
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, fmt.Errorf("timeout")
			}
			return nil, err
		}
		return nil, nil
	}

	// done 非 nil：循环读取直到 done 返回 true
	buf := make([]byte, maxRead)
	result := make([]byte, 0, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
			if len(result) >= maxRead {
				return result[:maxRead], nil
			}
			if done(result) {
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
