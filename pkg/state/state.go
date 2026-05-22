package state

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/tsmask/go-oam/pkg/cmd"
)

// SysInfo 系统基本信息
type SysInfo struct {
	Platform        string `json:"platform"`        // 操作系统平台
	PlatformVersion string `json:"platformVersion"` // 平台版本
	Arch            string `json:"arch"`            // 内核架构
	ArchVersion     string `json:"archVersion"`     // 内核版本
	OS              string `json:"os"`              // 操作系统类型
	Hostname        string `json:"hostname"`        // 主机名
	BootTime        int64  `json:"bootTime"`        // 系统启动时长，单位秒
	ProcessID       int    `json:"processId"`       // 当前进程 PID
	RunArch         string `json:"runArch"`         // Go 运行时架构
	RunVersion      string `json:"runVersion"`      // Go 运行时版本
	RunTime         int64  `json:"runTime"`         // 进程运行时长，单位秒
}

// SystemInfo 获取系统基本信息
func SystemInfo() SysInfo {
	info, err := host.Info()
	if err != nil {
		info = &host.InfoStat{}
	}
	return SysInfo{
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		Arch:            info.KernelArch,
		ArchVersion:     info.KernelVersion,
		OS:              info.OS,
		Hostname:        info.Hostname,
		BootTime:        int64(time.Since(time.Unix(int64(info.BootTime), 0)).Seconds()),
		ProcessID:       os.Getpid(),
		RunArch:         runtime.GOARCH,
		RunVersion:      runtime.Version(),
	}
}

// SystemSysTimeTime 系统时间信息
type SysTime struct {
	Current      string `json:"current"`      // 当前时间 "2006-01-02 15:04:05"
	Timezone     string `json:"timezone"`     // 时区偏移 "+0800 CST"
	TimezoneName string `json:"timezoneName"` // 时区名称 "CST"
	Timestamp    int64  `json:"timestamp"`    // 时间戳UTC毫秒
	RFC3339      string `json:"rfc3339"`      // RFC339格式 "2006-01-02T15:04:05Z07:00"
}

// SystemTime 获取当前系统时间
func SystemTime() SysTime {
	now := time.Now()
	return SysTime{
		Current:      now.Format(time.DateTime),
		Timezone:     now.Format("-0700 MST"),
		TimezoneName: now.Format("MST"),
		Timestamp:    now.UTC().UnixMilli(),
		RFC3339:      now.Format(time.RFC3339),
	}
}

// StateUName 获取系统内核描述信息。
// Linux 返回 "uname -a" 输出，Windows 返回 "{os} {platform} {version}"。
func StateUName() string {
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
	return strings.TrimSpace(uname)
}

// CpuUsage 进程与系统 CPU 使用信息
type CpuUsage struct {
	AppCpuUsage float64 `json:"appCpuUsage"` // 应用进程 CPU 使用率，单位 %
	SysCpuUsage float64 `json:"sysCpuUsage"` // 系统 CPU 使用率，单位 %
}

// MemUsage 进程与系统内存使用信息
type MemUsage struct {
	TotalMem    uint64  `json:"totalMem"`    // 系统总内存，单位字节
	AppUsedMem  uint64  `json:"appUsedMem"`  // 应用进程 RSS 内存，单位字节
	SysMemUsage float64 `json:"sysMemUsage"` // 系统内存使用率，单位 %
}

// StateProcUsage 获取指定进程的 CPU 与内存使用快照。
// pid 无效或采集失败时返回对应零值，不会中断整体采集。
func StateProcUsage(pid int32) (CpuUsage, MemUsage) {
	cpuUsage := CpuUsage{}
	memUsage := MemUsage{}

	p, err := process.NewProcess(pid)
	if err != nil {
		return cpuUsage, memUsage
	}

	if percent, err := p.CPUPercent(); err == nil {
		cpuUsage.AppCpuUsage = percent
	}
	if memInfo, err := p.MemoryInfo(); err == nil {
		memUsage.AppUsedMem = memInfo.RSS
	}

	// 系统级
	sysCpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(sysCpuPercents) > 0 {
		cpuUsage.SysCpuUsage = sysCpuPercents[0]
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		memUsage.TotalMem = vm.Total
		memUsage.SysMemUsage = vm.UsedPercent
	}

	return cpuUsage, memUsage
}
