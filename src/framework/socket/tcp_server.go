package socket

import (
	"fmt"
	"net"
	"strings"
)

// SocketTCP TCP服务端
type SocketTCP struct {
	Addr     string           `json:"addr"` // 主机地址
	Port     int64            `json:"port"` // 端口
	Listener *net.TCPListener `json:"listener"`
	StopChan chan struct{}    `json:"stop"` // 停止信号
}

// New 创建TCP服务端
func (s *SocketTCP) New() (*SocketTCP, error) {
	// IPV6地址协议
	proto := "tcp"
	if strings.Contains(s.Addr, ":") {
		proto = "tcp6"
		s.Addr = fmt.Sprintf("[%s]", s.Addr)
	}
	address := fmt.Sprintf("%s:%d", s.Addr, s.Port)

	// 解析 TCP 地址
	tcpAddr, err := net.ResolveTCPAddr(proto, address)
	if err != nil {
		return nil, err
	}

	// 监听 TCP 地址
	listener, err := net.ListenTCP(proto, tcpAddr)
	if err != nil {
		return nil, err
	}

	s.Listener = listener
	s.StopChan = make(chan struct{}, 1)
	return s, nil
}

// Close 关闭当前TCP服务端
func (s *SocketTCP) Close() {
	if s.Listener != nil {
		s.StopChan <- struct{}{}
		(*s.Listener).Close()
	}
}

// Resolve 处理消息
func (s *SocketTCP) Resolve(callback func(conn *net.Conn, err error)) {
	if s.Listener == nil {
		callback(nil, fmt.Errorf("tcp service not created"))
		return
	}

	defer func() {
		if err := recover(); err != nil {
			callback(nil, fmt.Errorf("tcp service panic err"))
		}
	}()

	listener := *s.Listener
	for {
		select {
		case <-s.StopChan:
			callback(nil, fmt.Errorf("udp service stop"))
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			defer conn.Close()
			callback(&conn, nil)
		}
	}
}
