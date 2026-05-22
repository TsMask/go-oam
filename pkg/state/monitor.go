package state

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// MonitorCPUMemUsage CPU 与内存的周期采样快照
type MonitorCPUMemUsage struct {
	CreateTime int64   `json:"createTime"` // 采样时间戳，单位毫秒
	CPU        float64 `json:"cpu"`        // 系统 CPU 使用率，单位 %
	LoadUsage  float64 `json:"loadUsage"`  // CPU 负载率（load1 / 核心数）
	CPULoad1   float64 `json:"cpuLoad1"`   // 1 分钟平均负载
	CPULoad5   float64 `json:"cpuLoad5"`   // 5 分钟平均负载
	CPULoad15  float64 `json:"cpuLoad15"`  // 15 分钟平均负载
	Memory     float64 `json:"memory"`     // 系统内存使用率，单位 %
}

// MonitorDiskIO 单块磁盘在采样周期内的 IO 差值
type MonitorDiskIO struct {
	CreateTime int64  `json:"createTime"` // 采样时间戳，单位毫秒
	Name       string `json:"name"`       // 磁盘设备名（如 sda）
	Read       uint64 `json:"read"`       // 周期内读取字节数
	Write      uint64 `json:"write"`      // 周期内写入字节数
	Count      uint64 `json:"count"`      // 周期内读写操作总次数
	Time       uint64 `json:"time"`       // 周期内 IO 耗时，单位毫秒
}

// MonitorNetIO 单块网卡在采样周期内的流量差值
type MonitorNetIO struct {
	CreateTime int64  `json:"createTime"` // 采样时间戳，单位毫秒
	Name       string `json:"name"`       // 网卡名称（如 eth0）
	Up         uint64 `json:"up"`         // 周期内发送字节数
	Down       uint64 `json:"down"`       // 周期内接收字节数
}

// LoadCPUMemUsage 采样 CPU 使用率、系统负载和内存使用率。
// duration 为 CPU 采样窗口（传 0 表示瞬时值），期间会阻塞等待。
func LoadCPUMemUsage(duration time.Duration) MonitorCPUMemUsage {
	result := MonitorCPUMemUsage{
		CreateTime: time.Now().UnixMilli(),
	}

	// 系统负载
	loadInfo, err := load.Avg()
	if err != nil {
		return result
	}
	result.CPULoad1 = loadInfo.Load1
	result.CPULoad5 = loadInfo.Load5
	result.CPULoad15 = loadInfo.Load15

	// CPU 使用率（duration 窗口内的平均值）
	totalPercent, err := cpu.Percent(duration, false)
	if err == nil && len(totalPercent) > 0 {
		result.CPU = totalPercent[0]
	}

	// 负载率 = load1 / 逻辑核心数
	if cpuCount, err := cpu.Counts(false); err == nil && cpuCount > 0 {
		result.LoadUsage = loadInfo.Load1 / float64(cpuCount)
	}

	// 内存使用率
	if memInfo, err := mem.VirtualMemory(); err == nil {
		result.Memory = memInfo.UsedPercent
	}

	return result
}

// LoadDiskIO 采样各磁盘在 duration 周期内的 IO 差值。
// 实现方式：先采集一次快照，等待 duration，再采集一次，计算差值。
func LoadDiskIO(duration time.Duration) []MonitorDiskIO {
	snapshot1, _ := disk.IOCounters()
	time.Sleep(duration)
	snapshot2, _ := disk.IOCounters()

	result := make([]MonitorDiskIO, 0, len(snapshot2))
	now := time.Now().UnixMilli()

	for name2, io2 := range snapshot2 {
		io1, ok := snapshot1[name2]
		if !ok {
			continue
		}
		item := MonitorDiskIO{
			CreateTime: now,
			Name:       io1.Name,
		}
		// 读写字节数差值
		if io2.ReadBytes > io1.ReadBytes {
			item.Read = io2.ReadBytes - io1.ReadBytes
		}
		if io2.WriteBytes > io1.WriteBytes {
			item.Write = io2.WriteBytes - io1.WriteBytes
		}
		// 读写次数累加
		if io2.ReadCount > io1.ReadCount {
			item.Count = io2.ReadCount - io1.ReadCount
		}
		if io2.WriteCount > io1.WriteCount {
			item.Count += io2.WriteCount - io1.WriteCount
		}
		// 读写耗时累加
		if io2.ReadTime > io1.ReadTime {
			item.Time = io2.ReadTime - io1.ReadTime
		}
		if io2.WriteTime > io1.WriteTime {
			item.Time += io2.WriteTime - io1.WriteTime
		}
		result = append(result, item)
	}
	return result
}

// LoadNetIO 采样各网卡在 duration 周期内的上下行流量差值，包括虚拟接口。
// 实现方式：先采集一次快照，等待 duration，再采集一次，计算差值。
func LoadNetIO(duration time.Duration) []MonitorNetIO {
	// 采集前：各网卡 + 汇总
	before := appendNetIO(nil)
	time.Sleep(duration)
	// 采集后
	after := appendNetIO(nil)

	result := make([]MonitorNetIO, 0, len(after))
	now := time.Now().UnixMilli()

	for name2, io2 := range after {
		io1, ok := before[name2]
		if !ok {
			continue
		}
		item := MonitorNetIO{
			CreateTime: now,
			Name:       io1.Name,
		}
		if io2.BytesSent > io1.BytesSent {
			item.Up = io2.BytesSent - io1.BytesSent
		}
		if io2.BytesRecv > io1.BytesRecv {
			item.Down = io2.BytesRecv - io1.BytesRecv
		}
		result = append(result, item)
	}
	return result
}

// appendNetIO 收集所有网卡（各接口 + 汇总）的 IO 计数器，返回 name -> stat 映射。
func appendNetIO(_ any) map[string]net.IOCountersStat {
	m := make(map[string]net.IOCountersStat)
	if perNic, err := net.IOCounters(true); err == nil {
		for _, s := range perNic {
			m[s.Name] = s
		}
	}
	if all, err := net.IOCounters(false); err == nil {
		for _, s := range all {
			m[s.Name] = s
		}
	}
	return m
}
