package state

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v4/process"
)

// PsProcessData 单个进程的运行时快照
type PsProcessData struct {
	PID            int32  `json:"pid"`            // 进程 PID
	Name           string `json:"name"`           // 进程名称
	PPID           int32  `json:"ppid"`           // 父进程 PID
	Username       string `json:"username"`       // 启动用户
	Status         string `json:"status"`         // 进程状态（running/sleeping/stopped 等，逗号分隔）
	StartTime      int64  `json:"startTime"`      // 进程启动时间戳，单位毫秒
	NumThreads     int32  `json:"numThreads"`     // 线程数
	NumConnections int    `json:"numConnections"` // 网络连接数
	CpuPercent     string `json:"cpuPercent"`     // CPU 使用率，单位 %
	DiskRead       uint64 `json:"diskRead"`       // 累计磁盘读取字节数
	DiskWrite      uint64 `json:"diskWrite"`      // 累计磁盘写入字节数
	Rss            uint64 `json:"rss"`            // 常驻内存，单位字节
	VMS            uint64 `json:"vms"`            // 虚拟内存，单位字节
	HWM            uint64 `json:"hwm"`            // 历史峰值内存，单位字节
	Data           uint64 `json:"data"`           // 数据段内存
	Stack          uint64 `json:"stack"`          // 栈内存
	Locked         uint64 `json:"locked"`         // 锁定内存
	Swap           uint64 `json:"swap"`           // 交换区占用
	CmdLine        string `json:"cmdLine"`        // 完整启动命令行
}

// PsProcessQuery 进程查询条件，各字段之间为 AND 关系，零值表示不过滤
type PsProcessQuery struct {
	PID      int32  `json:"pid"`      // 精确匹配 PID
	Name     string `json:"name"`     // 模糊匹配进程名（contains）
	Username string `json:"username"` // 模糊匹配用户名（contains）
}

// Processes 获取系统进程列表，按条件过滤并返回。
// 内部使用 4 个 goroutine 并发采集进程详情，结果按 PID 升序排列。
func Processes(query PsProcessQuery) ([]PsProcessData, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var (
		result     = make([]PsProcessData, 0, len(procs)/2)
		mu         sync.Mutex
		wg         sync.WaitGroup
		numWorkers = 4
	)

	// 将进程列表均分给 numWorkers 个 goroutine
	chunkSize := (len(procs) + numWorkers - 1) / numWorkers
	for i := range numWorkers {
		wg.Add(1)
		start := i * chunkSize
		end := min((i+1)*chunkSize, len(procs))

		go func(start, end int) {
			defer wg.Done()
			local := make([]PsProcessData, 0, end-start)
			for j := start; j < end; j++ {
				if data, ok := buildProcessData(procs[j], query); ok {
					local = append(local, data)
				}
			}
			mu.Lock()
			result = append(result, local...)
			mu.Unlock()
		}(start, end)
	}
	wg.Wait()

	sort.Slice(result, func(i, j int) bool {
		return result[i].PID < result[j].PID
	})

	return result, nil
}

// buildProcessData 采集单个进程的详细信息，按 query 条件过滤。
// 不满足条件时返回 false，调用方应丢弃该条数据。
func buildProcessData(proc *process.Process, query PsProcessQuery) (PsProcessData, bool) {
	data := PsProcessData{PID: proc.Pid}

	// 先采集 name/username 用于过滤判断
	if name, err := proc.Name(); err == nil {
		data.Name = name
	}
	if user, err := proc.Username(); err == nil {
		data.Username = user
	}

	// 过滤：PID 精确、进程名模糊、用户名模糊
	if query.PID > 0 && data.PID != query.PID {
		return data, false
	}
	if query.Name != "" && !strings.Contains(data.Name, query.Name) {
		return data, false
	}
	if query.Username != "" && !strings.Contains(data.Username, query.Username) {
		return data, false
	}

	// 基本信息
	data.PPID, _ = proc.Ppid()
	if statuses, err := proc.Status(); err == nil && len(statuses) > 0 {
		data.Status = strings.Join(statuses, ",")
	}
	if ts, err := proc.CreateTime(); err == nil {
		data.StartTime = ts
	}
	data.NumThreads, _ = proc.NumThreads()
	if conns, err := proc.Connections(); err == nil {
		data.NumConnections = len(conns)
	}

	// CPU
	cpuPercent, _ := proc.CPUPercent()
	data.CpuPercent = fmt.Sprintf("%.2f", cpuPercent)

	// 内存
	if memInfo, err := proc.MemoryInfo(); err == nil {
		data.Rss = memInfo.RSS
		data.VMS = memInfo.VMS
		data.HWM = memInfo.HWM
		data.Data = memInfo.Data
		data.Stack = memInfo.Stack
		data.Locked = memInfo.Locked
		data.Swap = memInfo.Swap
	}

	// 磁盘 IO
	if io, err := proc.IOCounters(); err == nil {
		data.DiskRead = io.ReadBytes
		data.DiskWrite = io.WriteBytes
	}

	// 启动命令
	data.CmdLine, _ = proc.Cmdline()

	return data, true
}
