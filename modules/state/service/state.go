package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/utils/parse"
	"github.com/tsmask/go-oam/modules/callback"
	"github.com/tsmask/go-oam/modules/state/model"
)

// 实例化服务层 State 结构体
var NewState = &State{}

// State 服务器系统相关信息 服务层处理
type State struct{}

// Info 系统信息
func (s *State) Info() model.State {
	state := model.State{
		OsInfo:    getUnameStr(),
		IpAddr:    getIPAddr(),
		Standby:   callback.Standby(),
		DiskSpace: getDiskSpace(),
	}

	hostName, err := os.Hostname()
	if err != nil {
		hostName = ""
	}
	state.HostName = hostName

	var pid int32 = 0
	neConf, ok := config.Get("ne").(map[string]any)
	if ok {
		state.Version = fmt.Sprint(neConf["version"])
		state.SerialNum = fmt.Sprint(neConf["serialnum"])
		state.ExpiryDate = fmt.Sprint(neConf["expirydate"])
		state.Capability = parse.Number(neConf["uenumber"])
		pid = int32(parse.Number(neConf["pid"]))
	}
	if pid == 0 {
		pid = int32(os.Getpid())
	}
	memUsage, cpuUsage := getMemAndCPUUsage(pid)
	state.CpuUsage = cpuUsage
	state.MemUsage = memUsage
	return state
}

// 获取主机的 IP 地址列表
func getIPAddr() []string {
	ipAddrs := []string{}
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			name := iface.Name
			if name[len(name)-1] == '0' {
				name = name[0 : len(name)-1]
				name = strings.Trim(name, "")
			}
			// ignore localhost
			if name == "lo" {
				continue
			}
			var addrs []string
			for _, v := range iface.Addrs {
				addrV := strings.Split(v.Addr, "/")[0]
				if strings.Contains(addrV, "::") {
					addrs = append(addrs, addrV)
				}
				if strings.Contains(addrV, ".") {
					addrs = append(addrs, addrV)
				}
			}
			ipAddrs = append(ipAddrs, addrs...)
		}
	}
	return ipAddrs
}

// getUnameStr Liunx uname -a
func getUnameStr() string {
	if runtime.GOOS == "windows" {
		info, err := host.Info()
		if err != nil {
			info.Platform = err.Error()
		}
		if err != nil {
			return ""
		}
		return info.PlatformVersion
	}
	osInfo, err := cmd.Exec("uname -a")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(osInfo)
}

// getMemAndCPUUsage 获取进程的内存和CPU占用率
func getMemAndCPUUsage(pid int32) (model.MemUsage, model.CpuUsage) {
	m := model.MemUsage{}
	c := model.CpuUsage{}

	// system RAM(KB)
	sysRam, err := mem.VirtualMemory()
	if err != nil {
		m.TotalMem = 0
		m.SysMemUsage = 0
	} else {
		m.TotalMem = sysRam.Total / 1024
		m.SysMemUsage = uint64(sysRam.UsedPercent * 100)
	}
	// system cpu percent
	totalPercent, err := cpu.Percent(300*time.Millisecond, true)
	if err != nil {
		c.SysCpuUsage = 0
	} else {
		var sum float64
		for _, corePercent := range totalPercent {
			sum += corePercent
		}
		// 计算平均使用率
		avgPercent := sum / float64(len(totalPercent))
		c.SysCpuUsage = uint16(avgPercent * 100)
	}

	// 根据PID取得进程资源
	proc, err := process.NewProcess(pid)
	if err != nil {
		return m, c
	}
	// pid RAM(KB)
	myRam, err := proc.MemoryInfo()
	if err != nil {
		m.NfUsedMem = 0
	} else {
		m.NfUsedMem = myRam.RSS / 1024
	}
	// pid cpu percent
	percent, err := proc.CPUPercent()
	if err != nil {
		c.NfCpuUsage = 0
	} else {
		c.NfCpuUsage = uint16(percent * 100)
	}
	return m, c
}

// getDiskSpace 获取磁盘空间信息
func getDiskSpace() model.DiskSpace {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil && err != context.DeadlineExceeded {
		return model.DiskSpace{}
	}

	ds := model.DiskSpace{}
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}
		ds.PartitionNum++
		ds.PartitionInfo = append(ds.PartitionInfo, model.PartitionInfo{
			Total:  usage.Total / 1024 / 1024,
			Used:   usage.Used / 1024 / 1024,
			Device: partition.Device,
		})
	}
	return ds
}
