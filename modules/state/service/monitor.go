package service

import (
	"time"

	"github.com/tsmask/go-oam/modules/state/model"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// 实例化服务层 Monitor 结构体
var NewMonitor = &Monitor{}

// Monitor 机器资源相关信息 服务层处理
type Monitor struct{}

// LoadCPUMem CPU内存使用率
func (s *Monitor) LoadCPUMem(duration time.Duration) model.MonitorBase {
	var itemBase model.MonitorBase
	itemBase.CreateTime = time.Now().UnixMilli()

	loadInfo, _ := load.Avg()
	itemBase.CPULoad1 = loadInfo.Load1
	itemBase.CPULoad5 = loadInfo.Load5
	itemBase.CPULoad15 = loadInfo.Load15
	totalPercent, _ := cpu.Percent(duration, false)
	if len(totalPercent) > 0 {
		itemBase.CPU = totalPercent[0]
	}
	if cpuCount, _ := cpu.Counts(false); cpuCount > 0 {
		itemBase.LoadUsage = loadInfo.Load1 / float64(cpuCount)
	} else {
		itemBase.LoadUsage = 0
	}

	memoryInfo, _ := mem.VirtualMemory()
	itemBase.Memory = memoryInfo.UsedPercent
	return itemBase
}

// LoadDiskIO 磁盘读写
func (s *Monitor) LoadDiskIO(duration time.Duration) []model.MonitorIO {
	ioStat, _ := disk.IOCounters()

	time.Sleep(duration)

	ioStat2, _ := disk.IOCounters()
	var ioList []model.MonitorIO
	timeMilli := time.Now().UnixMilli()
	for _, io2 := range ioStat2 {
		for _, io1 := range ioStat {
			if io2.Name == io1.Name {
				var itemIO model.MonitorIO
				itemIO.CreateTime = timeMilli
				itemIO.Name = io1.Name

				if io2.ReadBytes != 0 && io1.ReadBytes != 0 && io2.ReadBytes > io1.ReadBytes {
					itemIO.Read = io2.ReadBytes - io1.ReadBytes
				}
				if io2.WriteBytes != 0 && io1.WriteBytes != 0 && io2.WriteBytes > io1.WriteBytes {
					itemIO.Write = io2.WriteBytes - io1.WriteBytes
				}

				if io2.ReadCount != 0 && io1.ReadCount != 0 && io2.ReadCount > io1.ReadCount {
					itemIO.Count = io2.ReadCount - io1.ReadCount
				}
				if io2.WriteCount != 0 && io1.WriteCount != 0 && io2.WriteCount > io1.WriteCount {
					itemIO.Count += io2.WriteCount - io1.WriteCount
				}

				if io2.ReadTime != 0 && io1.ReadTime != 0 && io2.ReadTime > io1.ReadTime {
					itemIO.Time = io2.ReadTime - io1.ReadTime
				}
				if io2.WriteTime != 0 && io1.WriteTime != 0 && io2.WriteTime > io1.WriteTime {
					itemIO.Time += io2.WriteTime - io1.WriteTime
				}
				ioList = append(ioList, itemIO)
				break
			}
		}
	}
	return ioList
}

// LoadNetIO 网络接口（包括虚拟接口）
func (s *Monitor) LoadNetIO(duration time.Duration) []model.MonitorNetwork {
	// 获取当前时刻
	netStat, _ := net.IOCounters(true)
	netStatAll, _ := net.IOCounters(false)
	var netStatList []net.IOCountersStat
	netStatList = append(netStatList, netStat...)
	netStatList = append(netStatList, netStatAll...)

	time.Sleep(duration)

	// 获取结束时刻
	netStat2, _ := net.IOCounters(true)
	netStat2All, _ := net.IOCounters(false)
	var netStat2List []net.IOCountersStat
	netStat2List = append(netStat2List, netStat2...)
	netStat2List = append(netStat2List, netStat2All...)

	var netList []model.MonitorNetwork
	timeMilli := time.Now().UnixMilli()
	for _, net2 := range netStat2List {
		for _, net1 := range netStatList {
			if net2.Name == net1.Name {
				var itemNet model.MonitorNetwork
				itemNet.CreateTime = timeMilli
				itemNet.Name = net1.Name

				// 如果结束时刻发送字节数和当前时刻发送字节数都不为零，并且结束时刻发送字节数大于当前时刻发送字节数
				if net2.BytesSent != 0 && net1.BytesSent != 0 && net2.BytesSent > net1.BytesSent {
					itemNet.Up = net2.BytesSent - net1.BytesSent
				}
				if net2.BytesRecv != 0 && net1.BytesRecv != 0 && net2.BytesRecv > net1.BytesRecv {
					itemNet.Down = net2.BytesRecv - net1.BytesRecv
				}
				netList = append(netList, itemNet)
				break
			}
		}
	}

	return netList
}
