package ssh

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	gossh "golang.org/x/crypto/ssh"
)

// Session 远程交互式终端会话
//
// 通过 Client.NewSession 创建，提供 Write/Read/Close/WindowChange 四个基础方法。
type Session struct {
	session   *gossh.Session
	stdin     io.WriteCloser
	output    chan []byte
	closeOnce sync.Once
}

// NewSession 创建远程交互式终端会话
func (c *Client) NewSession(cols, rows int) (*Session, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("ssh: 连接已关闭")
	}

	sshSession, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh: 创建会话失败: %w", err)
	}

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		return nil, fmt.Errorf("ssh: 获取标准输入管道失败: %w", err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		return nil, fmt.Errorf("ssh: 获取标准输出管道失败: %w", err)
	}

	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		sshSession.Close()
		return nil, fmt.Errorf("ssh: 请求伪终端失败: %w", err)
	}
	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		return nil, fmt.Errorf("ssh: 启动Shell失败: %w", err)
	}

	s := &Session{
		session: sshSession,
		stdin:   stdin,
		output:  make(chan []byte, 4096),
	}

	// 后台读取远程输出到 channel
	go s.readOutput(stdout)

	return s, nil
}

// Close 关闭会话
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		if s.stdin != nil {
			s.stdin.Close()
		}
		if s.session != nil {
			s.session.Close()
		}
	})
}

// Write 写入命令（回车 \n 才会执行）
func (s *Session) Write(cmd string) (int, error) {
	if s.stdin == nil {
		return 0, fmt.Errorf("ssh session: 会话已关闭")
	}
	return s.stdin.Write([]byte(cmd))
}

// Read 读取输出，阻塞直到有数据到达
func (s *Session) Read() []byte {
	// 阻塞等待第一块数据
	data, ok := <-s.output
	if !ok {
		return nil
	}

	// 已有数据，排空 channel
	var buf bytes.Buffer
	buf.Write(data)
	for {
		select {
		case data, ok := <-s.output:
			if !ok {
				return buf.Bytes()
			}
			buf.Write(data)
		default:
			return buf.Bytes()
		}
	}
}

// WindowChange 调整终端窗口大小
func (s *Session) WindowChange(cols, rows int) error {
	if s.session == nil {
		return fmt.Errorf("ssh session: 会话已关闭")
	}
	return s.session.WindowChange(rows, cols)
}

// readOutput 后台读取远程输出到 channel
func (s *Session) readOutput(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			s.output <- data
		}
		if err != nil {
			close(s.output)
			return
		}
	}
}
