package model

import "github.com/shirou/gopsutil/v4/net"

// NetConnectData 网络连接进程数据
type NetConnectData struct {
	Type   string   `json:"type"`
	Status string   `json:"status"`
	Laddr  net.Addr `json:"localAddr"`
	Raddr  net.Addr `json:"remoteAddr"`
	PID    int32    `json:"pid"`
	Name   string   `json:"name"`
}

// NetConnectQuery 网络连接进程查询
type NetConnectQuery struct {
	Port int32  `json:"port"`
	Name string `json:"name"`
	PID  int32  `json:"pid"`
}
