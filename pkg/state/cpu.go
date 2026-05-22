package state

import (
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
)

// CPU 系统 CPU 信息
type CPU struct {
	Model    string    `json:"model"`    // CPU 型号
	Speed    float64   `json:"speed"`    // CPU 主频，单位 MHz
	Core     int       `json:"core"`     // 逻辑核心数
	CoreUsed []float64 `json:"coreUsed"` // 各核心使用率，单位 %
}

// SystemCPU 获取 CPU 型号、主频及各核心使用率
func SystemCPU() CPU {
	result := CPU{Core: runtime.NumCPU()}
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		result.Model = strings.TrimSpace(cpuInfo[0].ModelName)
		result.Speed = cpuInfo[0].Mhz
	}
	cpuPercent, err := cpu.Percent(0, true)
	if err == nil {
		result.CoreUsed = cpuPercent
	}
	return result
}
