package telnet

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Server telnet 服务端
type Server struct {
	Addr     string      // 监听地址，空字符串表示 0.0.0.0
	Port     string      // 监听端口
	OnError  func(error) // 可选：错误回调（handler panic、accept 异常等）
	MaxConns int         // 可选：最大并发连接数，0 表示不限制

	listener *net.TCPListener
	stopCh   chan struct{}
	stopOnce sync.Once
	sem      chan struct{} // 连接数信号量
}

// Listen 启动 telnet 监听
func (s *Server) Listen() error {
	// IPv6 地址协议
	proto := "tcp"
	if strings.Contains(s.Addr, ":") {
		proto = "tcp6"
	}

	tcpAddr, err := net.ResolveTCPAddr(proto, net.JoinHostPort(s.Addr, s.Port))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(proto, tcpAddr)
	if err != nil {
		return err
	}

	s.listener = listener
	s.stopCh = make(chan struct{})
	if s.MaxConns > 0 {
		s.sem = make(chan struct{}, s.MaxConns)
	}
	return nil
}

// Close 优雅关闭服务端
func (s *Server) Close() {
	s.stopOnce.Do(func() {
		if s.listener != nil {
			close(s.stopCh)
			s.listener.Close()
		}
	})
}

// ListenAddr 返回实际监听地址（适用于端口为 0 的场景）
func (s *Server) ListenAddr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// Accept 接受连接并分发给 handler（每个连接启动一个 goroutine）
// 阻塞运行直到 Close()，返回 nil 表示正常停止
// handler panic 通过 OnError 回调通知，不影响服务端运行
func (s *Server) Accept(handler func(net.Conn)) error {
	if s.listener == nil {
		return fmt.Errorf("telnet server not started")
	}

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return nil
			default:
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}

		// 并发连接数限制
		if s.sem != nil {
			select {
			case s.sem <- struct{}{}:
			default:
				conn.Close()
				s.onError(fmt.Errorf("telnet max conns reached: %d", s.MaxConns))
				continue
			}
		}

		go func() {
			defer conn.Close()
			if s.sem != nil {
				defer func() { <-s.sem }()
			}
			defer func() {
				if r := recover(); r != nil {
					s.onError(fmt.Errorf("telnet handler panic: %v", r))
				}
			}()
			handler(conn)
		}()
	}
}

// onError 通知错误（安全调用）
func (s *Server) onError(err error) {
	if s.OnError != nil {
		s.OnError(err)
	}
}
