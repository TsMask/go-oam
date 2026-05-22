package state

import (
	"runtime"

	"github.com/shirou/gopsutil/v4/mem"
)

// Memory 系统内存详情，单位字节
type Memory struct {
	Usage     float64 `json:"usage"`     // 系统内存使用率，单位 %
	FreeMem   uint64  `json:"freeMem"`   // 可用内存
	TotalMem  uint64  `json:"totalMem"`  // 总内存
	RSS       uint64  `json:"rss"`       // Go 进程常驻内存
	HeapTotal uint64  `json:"heapTotal"` // Go 堆总大小
	HeapUsed  uint64  `json:"heapUsed"`  // Go 堆已用
	External  uint64  `json:"external"`  // Go 非堆内存
}

// SystemMemory 获取系统及 Go 进程内存详情
func SystemMemory() Memory {
	vm, err := mem.VirtualMemory()
	if err != nil {
		vm = &mem.VirtualMemoryStat{}
	}
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return Memory{
		Usage:     vm.UsedPercent,
		FreeMem:   vm.Available,
		TotalMem:  vm.Total,
		RSS:       memStats.Sys,
		HeapTotal: memStats.HeapSys,
		HeapUsed:  memStats.HeapAlloc,
		External:  memStats.Sys - memStats.HeapSys,
	}
}
