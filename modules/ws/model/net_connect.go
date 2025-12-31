package model

import "github.com/shirou/gopsutil/v4/net"

// NetConnectData 网络连接进程数据
type NetConnectData struct {
	Type   string   `json:"type"`   // 连接类型
	Status string   `json:"status"` // 连接状态
	Laddr  net.Addr `json:"localAddr"`
	Raddr  net.Addr `json:"remoteAddr"`
	PID    int32    `json:"pid"`  // 进程ID
	Name   string   `json:"name"` // 进程名称
}

// NetConnectQuery 网络连接进程查询
type NetConnectQuery struct {
	Port int32  `json:"port"` // 端口号
	Name string `json:"name"` // 进程名称
	PID  int32  `json:"pid"`  // 进程ID
}
