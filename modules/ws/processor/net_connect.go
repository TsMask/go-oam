package processor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/modules/ws/model"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// GetNetConnections 获取网络连接进程
func GetNetConnections(data any) ([]model.NetConnectData, error) {
	msgByte, _ := json.Marshal(data)
	var query model.NetConnectQuery
	if err := json.Unmarshal(msgByte, &query); err != nil {
		return nil, fmt.Errorf("query data structure error, %s", err.Error())
	}

	dataArr := []model.NetConnectData{}
	for _, netType := range [...]string{"tcp", "udp"} {
		connections, err := net.Connections(netType)
		if err != nil {
			continue
		}
		for _, conn := range connections {
			if query.PID > 0 && query.PID != conn.Pid {
				continue
			}
			proc, err := process.NewProcess(conn.Pid)
			if err == nil {
				name, err := proc.Name()
				if err != nil {
					continue
				}
				if query.Name != "" && !strings.Contains(name, query.Name) {
					continue
				}
				if query.Port > 0 && query.Port != int32(conn.Laddr.Port) && query.Port != int32(conn.Raddr.Port) {
					continue
				}
				dataArr = append(dataArr, model.NetConnectData{
					Type:   netType,
					Status: conn.Status,
					Laddr:  conn.Laddr,
					Raddr:  conn.Raddr,
					PID:    conn.Pid,
					Name:   name,
				})
			}
		}
	}

	return dataArr, nil
}
