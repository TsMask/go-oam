package state

import (
	"strings"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// NetConnectData 单条网络连接信息，关联到持有该连接的进程
type NetConnectData struct {
	Type   string   `json:"type"`       // 协议类型："tcp" 或 "udp"
	Status string   `json:"status"`     // 连接状态（如 ESTABLISHED、LISTEN）
	Laddr  net.Addr `json:"localAddr"`  // 本地地址（IP + 端口）
	Raddr  net.Addr `json:"remoteAddr"` // 远端地址（IP + 端口）
	PID    int32    `json:"pid"`        // 持有该连接的进程 PID
	Name   string   `json:"name"`       // 进程名称
}

// NetConnectQuery 网络连接查询条件，各字段之间为 AND 关系，零值表示不过滤
type NetConnectQuery struct {
	Port int32  `json:"port"` // 匹配本地或远端端口
	Name string `json:"name"` // 模糊匹配进程名（contains）
	PID  int32  `json:"pid"`  // 精确匹配进程 PID
}

// NetConnections 遍历系统所有 tcp/udp 连接，按条件过滤后返回。
// 每条连接会查询其所属进程信息，进程已退出的连接会被跳过。
// 多个查询条件之间为 AND 关系。
func NetConnections(query NetConnectQuery) ([]NetConnectData, error) {
	result := []NetConnectData{}

	for _, proto := range [2]string{"tcp", "udp"} {
		connections, err := net.Connections(proto)
		if err != nil {
			continue
		}
		for _, conn := range connections {
			// PID 精确过滤
			if query.PID > 0 && query.PID != conn.Pid {
				continue
			}
			// 获取进程信息，进程已退出则跳过
			proc, err := process.NewProcess(conn.Pid)
			if err != nil {
				continue
			}
			procName, err := proc.Name()
			if err != nil {
				continue
			}
			// 进程名模糊匹配
			if query.Name != "" && !strings.Contains(procName, query.Name) {
				continue
			}
			// 端口匹配本地或远端
			if query.Port > 0 && query.Port != int32(conn.Laddr.Port) && query.Port != int32(conn.Raddr.Port) {
				continue
			}
			result = append(result, NetConnectData{
				Type:   proto,
				Status: conn.Status,
				Laddr:  conn.Laddr,
				Raddr:  conn.Raddr,
				PID:    conn.Pid,
				Name:   procName,
			})
		}
	}
	return result, nil
}
