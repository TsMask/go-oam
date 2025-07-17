package processor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/modules/ws/model"

	"github.com/shirou/gopsutil/v4/process"
)

// GetProcessData 获取进程数据
func GetProcessData(data any) ([]model.PsProcessData, error) {
	msgByte, _ := json.Marshal(data)
	var query model.PsProcessQuery
	err := json.Unmarshal(msgByte, &query)
	if err != nil {
		logger.Warnf("ws processor GetNetConnections err: %s", err.Error())
		return nil, fmt.Errorf("query data structure error")
	}

	var processes []*process.Process
	processes, err = process.Processes()
	if err != nil {
		return nil, err
	}

	// 解析数据
	handleData := func(proc *process.Process) (model.PsProcessData, bool) {
		procData := model.PsProcessData{
			PID: proc.Pid,
		}
		if procName, err := proc.Name(); err == nil {
			procData.Name = procName
		}
		if username, err := proc.Username(); err == nil {
			procData.Username = username
		}

		// 查询过滤
		if query.PID > 0 && procData.PID != query.PID {
			return procData, false
		}
		if query.Name != "" && !strings.Contains(procData.Name, query.Name) {
			return procData, false
		}
		if query.Username != "" && !strings.Contains(procData.Username, query.Username) {
			return procData, false
		}

		procData.PPID, _ = proc.Ppid()
		if statusArray, err := proc.Status(); err == nil && len(statusArray) > 0 {
			procData.Status = strings.Join(statusArray, ",")
		}
		if createTime, err := proc.CreateTime(); err == nil {
			procData.StartTime = createTime
		}
		procData.NumThreads, _ = proc.NumThreads()
		if connections, err := proc.Connections(); err == nil {
			procData.NumConnections = len(connections)
		}
		cpuPercent, _ := proc.CPUPercent()
		procData.CpuPercent = fmt.Sprintf("%.2f", cpuPercent)
		menInfo, procErr := proc.MemoryInfo()
		if procErr == nil {
			procData.Rss = menInfo.RSS
			procData.Data = menInfo.Data
			procData.VMS = menInfo.VMS
			procData.HWM = menInfo.HWM
			procData.Stack = menInfo.Stack
			procData.Locked = menInfo.Locked
			procData.Swap = menInfo.Swap
		}
		if ioStat, err := proc.IOCounters(); err == nil {
			procData.DiskWrite = ioStat.WriteBytes
			procData.DiskRead = ioStat.ReadBytes
		}
		procData.CmdLine, _ = proc.Cmdline()

		return procData, true
	}

	var (
		dataArr    = []model.PsProcessData{}
		mu         sync.Mutex
		wg         sync.WaitGroup
		numWorkers = 4
	)

	chunkSize := (len(processes) + numWorkers - 1) / numWorkers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if end > len(processes) {
			end = len(processes)
		}

		go func(start, end int) {
			defer wg.Done()
			localDataArr := make([]model.PsProcessData, 0, end-start) // 本地切片避免竞态
			for j := start; j < end; j++ {
				if data, ok := handleData(processes[j]); ok {
					localDataArr = append(localDataArr, data)
				}
			}
			mu.Lock()
			dataArr = append(dataArr, localDataArr...)
			mu.Unlock()
		}(start, end)
	}

	wg.Wait()

	sort.Slice(dataArr, func(i, j int) bool {
		return dataArr[i].PID < dataArr[j].PID
	})

	return dataArr, err
}
