package telnet

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

// ClientSession Telnet客户端会话对象
type ClientSession struct {
	Addr        string        `json:"addr"`     // telnet地址
	Port        string        `json:"port"`     // telnet端口
	User        string        `json:"user"`     // 认证用户名
	Password    string        `json:"password"` // 认证密码
	DialTimeOut time.Duration // 连接超时断开，默认10秒
	conn        net.Conn      // 连接实例
}

// Close 关闭会话
func (s *ClientSession) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

// WindowChange informs the remote host about a terminal window dimension change to h rows and w columns.
func (s *ClientSession) WindowChange(h, w int) error {
	if s.conn == nil {
		return fmt.Errorf("client is nil to content write failed")
	}
	// 需要确保接收方理解并正确处理发送窗口大小设置命令
	s.conn.Write([]byte{255, 251, 31})
	s.conn.Write([]byte{255, 250, 31, byte(w >> 8), byte(w & 0xFF), byte(h >> 8), byte(h & 0xFF), 255, 240})
	return nil
}

// Write 写入命令 根据客户端情况不带回车(\n)也会执行
func (s *ClientSession) Write(cmd string) (int, error) {
	if s.conn == nil {
		return 0, fmt.Errorf("client is nil to content write failed")
	}
	return s.conn.Write([]byte(cmd))
}

// Read 读取结果 等待一会才有结果
func (s *ClientSession) Read() []byte {
	if s.conn == nil {
		return []byte{}
	}

	buf := make([]byte, 1024)
	// 设置读取超时时间为100毫秒
	s.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := s.conn.Read(buf)
	if err != nil {
		return []byte{}
	}
	return buf[:n]
}

// CombinedOutput 发送命令带结果返回
func (s *ClientSession) CombinedOutput(cmd string) (string, error) {
	n, err := s.Write(cmd)
	if n == 0 || err != nil {
		return "", err
	}

	var buf bytes.Buffer
	tmp := make([]byte, 1024)
	for {
		// 设置读取超时时间为1000毫秒
		s.conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
		n, err := s.conn.Read(tmp)
		if err != nil {
			// 判断是否是超时错误
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				break
			}
			break
		}
		if n == 0 {
			break
		}
		buf.Write(tmp[:n])
	}
	defer buf.Reset()
	return buf.String(), nil
}
