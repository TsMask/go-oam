package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// NewClientSession 创建本地Bash客户端会话对象
func NewClientSession(cols, rows int) (*LocalClientSession, error) {
	// Create arbitrary command.
	c := exec.Command("bash")

	// Start the command with a pty.
	ptmx, err := pty.StartWithSize(c, &pty.Winsize{
		Rows: uint16(rows), // ws_row: Number of rows (in cells).
		Cols: uint16(cols), // ws_col: Number of columns (in cells).
		X:    0,            // ws_xpixel: Width in pixels.
		Y:    0,            // ws_ypixel: Height in pixels.
	})
	if err != nil {
		return nil, err
	}
	return &LocalClientSession{
		Ptmx: ptmx,
	}, nil
}

// LocalClientSession 本地Bash客户端会话对象
type LocalClientSession struct {
	Ptmx *os.File
}

// Close 关闭会话
func (s *LocalClientSession) Close() {
	if s.Ptmx != nil {
		s.Ptmx.Close()
	}
}

// Write 写入命令 回车(\n)才会执行
func (s *LocalClientSession) Write(cmd string) (int, error) {
	if s.Ptmx == nil {
		return 0, fmt.Errorf("ssh client session is nil to content write failed")
	}
	return s.Ptmx.Write([]byte(cmd))
}

// Read 读取结果
func (s *LocalClientSession) Read() []byte {
	if s.Ptmx == nil {
		return []byte{}
	}
	// 读取并输出伪终端中的数据
	buffer := make([]byte, 1024)
	n, err := s.Ptmx.Read(buffer)
	if n == 0 || err != nil {
		return []byte{}
	}
	return buffer[:n]
}

// Read 读取结果
func (s *LocalClientSession) WindowChange(cols, rows int) {
	if s.Ptmx == nil {
		return
	}
	pty.Setsize(s.Ptmx, &pty.Winsize{
		Rows: uint16(rows), // ws_row: Number of rows (in cells).
		Cols: uint16(cols), // ws_col: Number of columns (in cells).
		X:    0,            // ws_xpixel: Width in pixels.
		Y:    0,            // ws_ypixel: Height in pixels.
	})
}
