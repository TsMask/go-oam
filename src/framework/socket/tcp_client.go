package socket

import (
	"bytes"
	"fmt"
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
		c.Addr = fmt.Sprintf("[%s]", c.Addr)
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

	tmp := make([]byte, 1024)
	for {
		select {
		case <-time.After(timeout):
			return buf.String(), fmt.Errorf("timeout")
		default:
			// 读取命令消息
			n, err := conn.Read(tmp)
			if n == 0 || err != nil {
				tmp = nil
				break
			}
			buf.Write(tmp[:n])
			tmpStr := string(tmp[:n])

			// 是否有终止符
			arr := []string{">", "#", "# ", "> "}
			for _, v := range arr {
				if strings.HasSuffix(tmpStr, v) {
					tmp = nil
					return buf.String(), nil
				}
			}
		}
	}
}
