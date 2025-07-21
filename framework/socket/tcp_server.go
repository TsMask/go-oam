package socket

import (
	"fmt"
	"net"
	"strings"
)

// ServerTCP TCP服务端
type ServerTCP struct {
	Addr     string           `json:"addr"` // 主机地址
	Port     string           `json:"port"` // 端口
	listener *net.TCPListener // 监听服务
	stopChan chan struct{}    // 停止信号
}

// Listen 创建TCP服务端
func (s *ServerTCP) Listen() error {
	// IPV6地址协议
	proto := "tcp"
	if strings.Contains(s.Addr, ":") {
		proto = "tcp6"
		s.Addr = fmt.Sprintf("[%s]", s.Addr)
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
	if s.listener != nil {
		s.stopChan <- struct{}{}
		s.listener.Close()
	}
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
				continue
			}
			defer conn.Close()
			callback(conn, nil)
		}
	}
}
