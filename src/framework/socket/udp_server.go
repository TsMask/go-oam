package socket

import (
	"fmt"
	"net"
	"strings"
)

// ServerUDP UDP服务端
type ServerUDP struct {
	Addr     string        `json:"addr"` // 主机地址
	Port     string        `json:"port"` // 端口
	conn     *net.UDPConn  // 监听服务
	stopChan chan struct{} // 停止信号
}

// Listen 创建UDP服务端
func (s *ServerUDP) Listen() error {
	// IPV6地址协议
	proto := "udp"
	if strings.Contains(s.Addr, ":") {
		proto = "udp6"
		s.Addr = fmt.Sprintf("[%s]", s.Addr)
	}
	address := net.JoinHostPort(s.Addr, s.Port)

	// 解析 UDP 地址
	udpAddr, err := net.ResolveUDPAddr(proto, address)
	if err != nil {
		return err
	}

	// 监听 UDP 地址
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	s.conn = conn
	s.stopChan = make(chan struct{}, 1)
	return nil
}

// ServerUDP 关闭当前UDP服务端
func (s *ServerUDP) Close() {
	if s.conn != nil {
		s.stopChan <- struct{}{}
		s.conn.Close()
	}
}

// Resolve 处理消息
func (s *ServerUDP) Resolve(callback func(*net.UDPConn, error)) {
	if s.conn == nil {
		callback(nil, fmt.Errorf("udp service not created"))
		return
	}

	defer func() {
		if err := recover(); err != nil {
			callback(nil, fmt.Errorf("udp service panic err"))
		}
	}()

	for {
		select {
		case <-s.stopChan:
			callback(nil, fmt.Errorf("udp service not created"))
			return
		default:
			callback(s.conn, nil)
		}
	}
}
