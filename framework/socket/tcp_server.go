package socket

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ServerTCP TCP服务端
type ServerTCP struct {
	Addr     string           `json:"addr"` // 主机地址
	Port     string           `json:"port"` // 端口
	listener *net.TCPListener // 监听服务
	stopChan chan struct{}    // 停止信号
	stopOnce sync.Once        // 停止一次
}

// Listen 创建TCP服务端
func (s *ServerTCP) Listen() error {
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

// Close 关闭当前TCP服务端
func (s *ServerTCP) Close() {
	s.stopOnce.Do(func() {
		if s.listener != nil {
			close(s.stopChan)
			s.listener.Close()
		}
	})
}

// Resolve 处理消息
func (s *ServerTCP) Resolve(callback func(conn net.Conn, err error)) {
	if s.listener == nil {
		callback(nil, fmt.Errorf("tcp service not created"))
		return
	}

	defer func() {
		if err := recover(); err != nil {
			callback(nil, fmt.Errorf("tcp service panic err"))
		}
	}()

	for {
		select {
		case <-s.stopChan:
			callback(nil, fmt.Errorf("tcp service stop"))
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				// 避免在 listener 关闭后进入紧密循环
				select {
				case <-s.stopChan:
					return
				default:
					// 如果是临时错误，休眠一小段时间
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				defer func() {
					if err := recover(); err != nil {
						// 记录 panic，避免 crashing 整个服务
						callback(nil, fmt.Errorf("tcp connection handler panic: %v", err))
					}
				}()
				callback(c, nil)
			}(conn)
		}
	}
}
