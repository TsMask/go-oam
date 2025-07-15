package telnet

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

// Client 客户端参数
type Client struct {
	Addr        string        `json:"addr"`     // telnet地址
	Port        string        `json:"port"`     // telnet端口
	User        string        `json:"user"`     // 认证用户名
	Password    string        `json:"password"` // 认证密码
	DialTimeOut time.Duration // 连接超时断开，默认10秒
	conn        net.Conn      // 连接实例
}

// New 客户端连接
func (c *Client) Connect() error {
	// IPV6地址协议
	proto := "tcp"
	if strings.Contains(c.Addr, ":") {
		proto = "tcp6"
	}
	address := net.JoinHostPort(c.Addr, c.Port)

	// 默认等待10s
	if c.DialTimeOut == 0 {
		c.DialTimeOut = 10 * time.Second
	}

	// 连接到 Telnet 服务器
	conn, err := net.DialTimeout(proto, address, c.DialTimeOut)
	if err != nil {
		return err
	}

	// 进行登录
	if c.User != "" {
		time.Sleep(100 * time.Millisecond)
		conn.Write([]byte(c.User + "\r\n"))
		// fmt.Fprintln(conn, c.User)
	}
	if c.Password != "" {
		time.Sleep(100 * time.Millisecond)
		conn.Write([]byte(c.Password + "\r\n"))
		// fmt.Fprintln(conn, c.Password)
	}

	c.conn = conn
	return nil
}

// Close 关闭当前客户端
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// WindowChange informs the remote host about a terminal window dimension change to h rows and w columns.
func (c *Client) WindowChange(h, w int) error {
	if c.conn == nil {
		return fmt.Errorf("telnet client not connected")
	}
	// 需要确保接收方理解并正确处理发送窗口大小设置命令
	c.conn.Write([]byte{255, 251, 31})
	c.conn.Write([]byte{255, 250, 31, byte(w >> 8), byte(w & 0xFF), byte(h >> 8), byte(h & 0xFF), 255, 240})
	return nil
}

// RunCMD 执行单次命令，根据终止符停止 ">", "#", "# ", "> "
func (c *Client) RunCMD(cmd string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("telnet client not connected")
	}

	// 写入命令
	if cmd != "" {
		if _, err := c.conn.Write([]byte(cmd)); err != nil {
			return "", err
		}
		time.Sleep(100 * time.Millisecond)
	}

	var buf bytes.Buffer
	tmp := make([]byte, 1024)
	for {
		// 读取命令消息
		n, err := c.conn.Read(tmp)
		if n == 0 || err != nil {
			tmp = nil
			break
		}

		tmpStr := string(tmp[:n])
		buf.WriteString(tmpStr)

		// 是否有终止符
		arr := []string{">", "#", "# ", "> "}
		for _, v := range arr {
			if strings.HasSuffix(tmpStr, v) {
				tmp = nil
				break
			}
		}
	}
	defer buf.Reset()
	return buf.String(), nil
}
