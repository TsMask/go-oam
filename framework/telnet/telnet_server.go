package telnet

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Server 服务参数
type Server struct {
	Addr     string           `json:"addr"` // telnet地址
	Port     string           `json:"port"` // telnet端口
	listener *net.TCPListener // 监听服务
	stopChan chan struct{}    // 停止信号
	stopOnce sync.Once        // 停止一次
}

// New 服务创建监听
func (s *Server) Listen() error {
	// IPV6地址协议
	proto := "tcp"
	if strings.Contains(s.Addr, ":") {
		proto = "tcp6"
	}
	address := net.JoinHostPort(s.Addr, s.Port)

	// 解析 TCP 地址
	tcpAddr, err := net.ResolveTCPAddr(proto, address)
	if err != nil {
		return err
	}

	// 监听 TCP 地址
	listener, err := net.ListenTCP(proto, tcpAddr)
	if err != nil {
		return err
	}

	s.listener = listener
	s.stopChan = make(chan struct{}, 1)
	return nil
}

// Close 关闭当前服务
func (s *Server) Close() {
	s.stopOnce.Do(func() {
		if s.listener != nil {
			close(s.stopChan)
			s.listener.Close()
		}
	})
}

// Resolve 处理消息
func (t *Server) Resolve(callback func(conn net.Conn, err error)) {
	if t.listener == nil {
		callback(nil, fmt.Errorf("telnet service not created"))
		return
	}

	defer func() {
		if err := recover(); err != nil {
			callback(nil, fmt.Errorf("telnet service panic err"))
		}
	}()

	for {
		select {
		case <-t.stopChan:
			callback(nil, fmt.Errorf("telnet service stop"))
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-t.stopChan:
					return
				default:
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				defer func() {
					if err := recover(); err != nil {
						callback(nil, fmt.Errorf("telnet connection handler panic: %v", err))
					}
				}()
				callback(c, nil)
			}(conn)
		}
	}
}
