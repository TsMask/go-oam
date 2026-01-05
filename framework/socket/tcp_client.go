package socket

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// ClientTCP 连接TCP客户端
type ClientTCP struct {
	Addr        string        `json:"addr"` // 主机地址
	Port        string        `json:"port"` // 端口
	DialTimeOut time.Duration // 连接超时断开, 默认5秒
	conn        net.Conn      // 客户端
}

// Connect 连接TCP客户端
func (c *ClientTCP) Connect() error {
	// IPV6地址协议
	proto := "tcp"
	if strings.Contains(c.Addr, ":") {
		proto = "tcp6"
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

// Close 关闭当前TCP客户端
func (c *ClientTCP) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Send 发送消息
func (c *ClientTCP) Send(msg []byte, timeout time.Duration) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("tcp client not connected")
	}
	conn := c.conn

	// 写入信息
	if len(msg) > 0 {
		if _, err := conn.Write(msg); err != nil {
			return "", err
		}
	}

	var buf bytes.Buffer
	defer buf.Reset()

	// 设置读取超时
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", fmt.Errorf("set read deadline error: %v", err)
	}

	tmp := make([]byte, 1024)
	for {
		// 读取命令消息
		n, err := conn.Read(tmp)
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return buf.String(), fmt.Errorf("timeout")
			}
			if err == io.EOF {
				break
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
