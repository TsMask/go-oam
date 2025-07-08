package socket

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

// ConnUDP 连接UDP客户端
type ConnUDP struct {
	Addr string `json:"addr"` // 主机地址
	Port int64  `json:"port"` // 端口

	DialTimeOut time.Duration `json:"dialTimeOut"` // 连接超时断开

	Client     *net.Conn `json:"client"`
	LastResult string    `json:"lastResult"` // 记最后一次发送消息的结果
}

// New 创建UDP客户端
func (c *ConnUDP) New() (*ConnUDP, error) {
	// IPV6地址协议
	proto := "udp"
	if strings.Contains(c.Addr, ":") {
		proto = "udp6"
		c.Addr = fmt.Sprintf("[%s]", c.Addr)
	}
	address := net.JoinHostPort(c.Addr, fmt.Sprint(c.Port))

	// 默认等待5s
	if c.DialTimeOut == 0 {
		c.DialTimeOut = 5 * time.Second
	}

	// 连接到服务端
	client, err := net.DialTimeout(proto, address, c.DialTimeOut)
	if err != nil {
		return nil, err
	}

	c.Client = &client
	return c, nil
}

// Close 关闭当前UDP客户端
func (c *ConnUDP) Close() {
	if c.Client != nil {
		(*c.Client).Close()
	}
}

// Send 发送消息
func (c *ConnUDP) Send(msg []byte, ms int) (string, error) {
	if c.Client == nil {
		return "", fmt.Errorf("udp client not connected")
	}
	conn := *c.Client

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
		case <-time.After(time.Duration(time.Duration(ms).Milliseconds())):
			c.LastResult = buf.String()
			return c.LastResult, fmt.Errorf("timeout")
		default:
			// 读取命令消息
			n, err := conn.Read(tmp)
			if n == 0 || err != nil {
				tmp = nil
				break
			}

			tmpStr := string(tmp[:n])
			buf.WriteString(tmpStr)

			// 是否有终止符
			if strings.HasSuffix(tmpStr, ">") || strings.HasSuffix(tmpStr, "> ") || strings.HasSuffix(tmpStr, "# ") {
				tmp = nil
				c.LastResult = buf.String()
				return c.LastResult, nil
			}
		}
	}
}
