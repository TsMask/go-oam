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
	"github.com/tsmask/go-oam/modules/callback"
	"github.com/tsmask/go-oam/modules/state/model"
)

func NewStateService() *State {
	return &State{}
}

// State 服务器系统相关信息 服务层处理
type State struct{}

// Info 系统信息
func (s *State) Info(cfg *config.Config, handler callback.CallbackHandler) model.State {
	state := model.State{
		OsInfo:    getUnameStr(),
		IpAddr:    getIPAddr(),
		Standby:   s.Standby(handler),
		DiskSpace: getDiskSpace(),
	}

	hostName, err := os.Hostname()
	if err != nil {
		hostName = ""
	}
	state.HostName = hostName
	var pid int32

	cfg.View(func(c *config.Config) {
		state.Version = c.NE.Version
		state.SerialNum = c.NE.SerialNum
		state.ExpiryDate = c.NE.ExpiryDate
		state.Capability = int64(c.NE.UeNumber)
		pid = int32(c.NE.Pid)
	})

	if pid != 0 {
		pid = int32(os.Getpid())
	}
	memUsage, cpuUsage := getMemAndCPUUsage(pid)
	state.CpuUsage = cpuUsage
	state.MemUsage = memUsage
	return state
}

// Standby 备用状态
func (s *State) Standby(handler callback.CallbackHandler) bool {
	if handler != nil {
		return handler.Standby()
	}
	return false
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
			return err.Error()
		}
		return fmt.Sprintf("%s %s %s", info.OS, info.Platform, info.PlatformVersion)
	}
	uname, err := cmd.Exec("uname -a")
	if err != nil {
		return ""
	}
	return uname
}

// getMemAndCPUUsage 获取内存和CPU使用率
func getMemAndCPUUsage(pid int32) (model.MemUsage, model.CpuUsage) {
	memUsage := model.MemUsage{}
	cpuUsage := model.CpuUsage{}

	p, err := process.NewProcess(pid)
	if err != nil {
		return memUsage, cpuUsage
	}

	// 进程 CPU 使用率
	if percent, err := p.CPUPercent(); err == nil {
		cpuUsage.NfCpuUsage = uint16(percent * 100)
	}

	// 进程内存使用量
	if memInfo, err := p.MemoryInfo(); err == nil {
		memUsage.NfUsedMem = memInfo.RSS / 1024 // KB
	}

	// 系统 CPU 使用率
	if sysCpuPercents, err := cpu.Percent(0, false); err == nil && len(sysCpuPercents) > 0 {
		cpuUsage.SysCpuUsage = uint16(sysCpuPercents[0] * 100)
	}

	// 系统内存使用率
	if vm, err := mem.VirtualMemory(); err == nil {
		memUsage.TotalMem = vm.Total / 1024 // KB
		memUsage.SysMemUsage = uint64(vm.UsedPercent * 100)
	}

	return memUsage, cpuUsage
}

// getDiskSpace 获取磁盘空间
func getDiskSpace() model.DiskSpace {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	ds := model.DiskSpace{}
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return ds
	}

	ds.PartitionNum = uint8(len(parts))
	for _, part := range parts {
		usage, err := disk.Usage(part.Mountpoint)
		if err == nil {
			ds.PartitionInfo = append(ds.PartitionInfo, model.PartitionInfo{
				Device: part.Device,
				Total:  usage.Total / 1024 / 1024, // MB
				Used:   usage.Used / 1024 / 1024,  // MB
			})
		}
	}
	return ds
}
