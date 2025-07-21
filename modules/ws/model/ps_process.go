package model

// PsProcessData 进程数据
type PsProcessData struct {
	PID            int32  `json:"pid"`
	Name           string `json:"name"`
	PPID           int32  `json:"ppid"`
	Username       string `json:"username"`
	Status         string `json:"status"`
	StartTime      int64  `json:"startTime"`
	NumThreads     int32  `json:"numThreads"`
	NumConnections int    `json:"numConnections"`
	CpuPercent     string `json:"cpuPercent"`

	DiskRead  uint64 `json:"diskRead"`
	DiskWrite uint64 `json:"diskWrite"`

	Rss    uint64 `json:"rss"`
	VMS    uint64 `json:"vms"`
	HWM    uint64 `json:"hwm"`
	Data   uint64 `json:"data"`
	Stack  uint64 `json:"stack"`
	Locked uint64 `json:"locked"`
	Swap   uint64 `json:"swap"`

	CmdLine string `json:"cmdLine"`
}

// PsProcessQuery 进程查询
type PsProcessQuery struct {
	PID      int32  `json:"pid"`
	Name     string `json:"name"`
	Username string `json:"username"`
}
