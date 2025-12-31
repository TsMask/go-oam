package model

// PsProcessData 进程数据
type PsProcessData struct {
	PID            int32  `json:"pid"`
	Name           string `json:"name"`           // 进程名称
	PPID           int32  `json:"ppid"`           // 父进程ID
	Username       string `json:"username"`       // 进程所属用户名
	Status         string `json:"status"`         // 进程状态
	StartTime      int64  `json:"startTime"`      // 进程启动时间
	NumThreads     int32  `json:"numThreads"`     // 线程数
	NumConnections int    `json:"numConnections"` // 连接数
	CpuPercent     string `json:"cpuPercent"`     // CPU占用率

	DiskRead  uint64 `json:"diskRead"`  // 磁盘读取字节数
	DiskWrite uint64 `json:"diskWrite"` // 磁盘写入字节数

	Rss    uint64 `json:"rss"`    // 驻留集大小（Resident Set Size）
	VMS    uint64 `json:"vms"`    // 虚拟内存大小（Virtual Memory Size）
	HWM    uint64 `json:"hwm"`    // 高水位标记（High Water Mark）
	Data   uint64 `json:"data"`   // 数据段大小（Data Segment Size）
	Stack  uint64 `json:"stack"`  // 栈段大小（Stack Segment Size）
	Locked uint64 `json:"locked"` // 锁定的内存大小（Locked Memory Size）
	Swap   uint64 `json:"swap"`   // 交换空间大小（Swap Space Size）

	CmdLine string `json:"cmdLine"` // 命令行参数
}

// PsProcessQuery 进程查询
type PsProcessQuery struct {
	PID      int32  `json:"pid"`      // 进程ID
	Name     string `json:"name"`     // 进程名称
	Username string `json:"username"` // 进程所属用户名
}
