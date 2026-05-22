package socket

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ServerUDP UDP 服务端
type ServerUDP struct {
	Addr     string      // 监听地址
	Port     string      // 监听端口
	OnError  func(error) // 可选：错误回调（handler panic、读取异常等）
	MaxConns int         // 可选：最大并发连接数，0 表示不限制

	conn     *net.UDPConn
	stopCh   chan struct{}
	stopOnce sync.Once
	sem      chan struct{} // handler 信号量
}

// Listen 启动 UDP 监听
func (s *ServerUDP) Listen() error {
	proto := "udp"
	addr := s.Addr
	if strings.Contains(addr, ":") {
		proto = "udp6"
		addr = fmt.Sprintf("[%s]", addr)
	}

	udpAddr, err := net.ResolveUDPAddr(proto, net.JoinHostPort(addr, s.Port))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP(proto, udpAddr)
	if err != nil {
		return err
	}

	s.conn = conn
	s.stopCh = make(chan struct{})
	if s.MaxConns > 0 {
		s.sem = make(chan struct{}, s.MaxConns)
	}
	return nil
}

// Close 优雅关闭服务端
func (s *ServerUDP) Close() {
	s.stopOnce.Do(func() {
		if s.conn != nil {
			close(s.stopCh)
			s.conn.Close()
		}
	})
}

// ListenAddr 返回实际监听地址
func (s *ServerUDP) ListenAddr() net.Addr {
	if s.conn == nil {
		return nil
	}
	return s.conn.LocalAddr()
}

// Serve 读取数据报并分发给 handler（每个数据报一个 goroutine）
// handler 可通过 conn 发送响应，阻塞运行直到 Close()，返回 nil 表示正常停止
// handler panic 通过 OnError 回调通知，不影响服务端运行
func (s *ServerUDP) Serve(handler func(conn *net.UDPConn, data []byte, addr *net.UDPAddr)) error {
	if s.conn == nil {
		return fmt.Errorf("udp server not started")
	}

	buf := make([]byte, 65536) // UDP 最大包长
	errCount := 0
	for {
		select {
		case <-s.stopCh:
			return nil
		default:
		}

		n, remoteAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-s.stopCh:
				return nil
			default:
			}
			errCount++
			s.onError(fmt.Errorf("udp read: %w", err))
			backoff := time.Duration(errCount*50) * time.Millisecond
			if backoff > time.Second {
				backoff = time.Second
			}
			time.Sleep(backoff)
			continue
		}
		errCount = 0

		if n == 0 {
			continue
		}

		// 并发 handler 数量限制
		if s.sem != nil {
			select {
			case s.sem <- struct{}{}:
			default:
				s.onError(fmt.Errorf("udp max conns reached: %d, dropping packet", s.MaxConns))
				continue
			}
		}

		data := make([]byte, n)
		copy(data, buf[:n])
		go func() {
			defer func() {
				if s.sem != nil {
					<-s.sem
				}
			}()
			defer func() {
				if r := recover(); r != nil {
					s.onError(fmt.Errorf("udp handler panic: %v", r))
				}
			}()
			handler(s.conn, data, remoteAddr)
		}()
	}
}

// onError 通知错误（安全调用）
func (s *ServerUDP) onError(err error) {
	if s.OnError != nil {
		s.OnError(err)
	}
}
