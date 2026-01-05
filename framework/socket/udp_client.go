package socket

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

// ClientUDP 连接UDP客户端
type ClientUDP struct {
	Addr        string        `json:"addr"` // 主机地址
	Port        string        `json:"port"` // 端口
	DialTimeOut time.Duration // 连接超时断开, 默认5秒
	conn        net.Conn      // 客户端
}

// Connect 连接UDP客户端
func (c *ClientUDP) Connect() error {
	// IPV6地址协议
	proto := "udp"
	if strings.Contains(c.Addr, ":") {
		proto = "udp6"
	}
	address := net.JoinHostPort(c.Addr, c.Port)

	// 默认等待5s
	if c.DialTimeOut == 0 {
		c.DialTimeOut = 5 * time.Second
	}

	// 连接到服务端
	conn, err := net.DialTimeout(proto, address, c.DialTimeOut)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// Close 关闭当前UDP客户端
func (c *ClientUDP) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Send 发送消息
func (c *ClientUDP) Send(msg []byte, timeout time.Duration) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("udp client not connected")
	}

	// 写入信息
	if len(msg) > 0 {
		if _, err := c.conn.Write(msg); err != nil {
			return "", err
		}
	}

	var buf bytes.Buffer
	defer buf.Reset()

	// 设置读取超时
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", fmt.Errorf("set read deadline error: %v", err)
	}

	tmp := make([]byte, 1024)
	for {
		// 读取命令消息
		n, err := c.conn.Read(tmp)
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return buf.String(), fmt.Errorf("timeout")
			}
			return buf.String(), err
		}
		if n == 0 {
			break
		}

		buf.Write(tmp[:n])
		tmpStr := string(tmp[:n])

		// 是否有终止符
		arr := []string{">", "#", "# ", "> "}
		for _, v := range arr {
			if strings.HasSuffix(tmpStr, v) {
				return buf.String(), nil
			}
		}
	}
	return buf.String(), nil
}
