package socket

import (
	"fmt"
	"net"
	"strings"
)

// SocketUDP UDP服务端
type SocketUDP struct {
	Addr     string        `json:"addr"` // 主机地址
	Port     int64         `json:"port"` // 端口
	Conn     *net.UDPConn  `json:"conn"`
	StopChan chan struct{} `json:"stop"` // 停止信号
}

// New 创建UDP服务端
func (s *SocketUDP) New() error {
	// IPV6地址协议
	proto := "udp"
	if strings.Contains(s.Addr, ":") {
		proto = "udp6"
		s.Addr = fmt.Sprintf("[%s]", s.Addr)
	}
	address := fmt.Sprintf("%s:%d", s.Addr, s.Port)

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

	s.Conn = conn
	s.StopChan = make(chan struct{}, 1)
	return nil
}

// CloseService 关闭当前UDP服务端
func (s *SocketUDP) Close() {
	if s.Conn != nil {
		s.StopChan <- struct{}{}
		(*s.Conn).Close()
	}
}

// Resolve 处理消息
func (s *SocketUDP) Resolve(callback func(*net.UDPConn, error)) {
	if s.Conn == nil {
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
		case <-s.StopChan:
			callback(nil, fmt.Errorf("udp service not created"))
			return
		default:
			callback(s.Conn, nil)
		}
	}
}
