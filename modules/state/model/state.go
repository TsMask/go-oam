package model

// State 网元状态
type State struct {
	HostName string `json:"hostName"` // linux命令: hostname
	OsInfo   string `json:"osInfo"`   // linux命令: uname -a

	IpAddr []string `json:"ipAddr"` // 网络的ipv4和ipv6列表

	Version    string `json:"version"`    // 软件版本信息: 16.1.1
	SerialNum  string `json:"serialNum"`  // 序列号 12345678
	ExpiryDate string `json:"expiryDate"` // 到期时间 YYYY-MM-DD
	Capability int64  `json:"capability"` // UE数量
	Standby    bool   `json:"standby"`    // 是否备用模式

	CpuUsage  CpuUsage  `json:"cpuUsage"`
	MemUsage  MemUsage  `json:"memUsage"`
	DiskSpace DiskSpace `json:"diskSpace"`
}

// DiskSpace 网元状态磁盘信息
type DiskSpace struct {
	PartitionNum  uint8           `json:"partitionNum"`  // 分区数量
	PartitionInfo []PartitionInfo `json:"partitionInfo"` // 磁盘分区信息
}

// PartitionInfo 网元状态磁盘使用信息
type PartitionInfo struct {
	Device string `json:"device"` // 驱动名称
	Total  uint64 `json:"total"`  // 总大小MB (v/1024).toFixed(2) GB
	Used   uint64 `json:"used"`   // 已用大小MB (v/1024).toFixed(2) GB
}

// MemUsage 网元状态内存信息
type MemUsage struct {
	TotalMem    uint64 `json:"totalMem"`    // 总内存KB (v/1024/1024).toFixed(2) GB
	NfUsedMem   uint64 `json:"nfUsedMem"`   // 进程内存KB (v/1024).toFixed(2) MB
	SysMemUsage uint64 `json:"sysMemUsage"` // 系统内存使用率 (v/100).toFixed(2) %
}

// CpuUsage 网元状态cpu信息
type CpuUsage struct {
	NfCpuUsage  uint16 `json:"nfCpuUsage"`  // 进程cpu使用率 (v/100).toFixed(2) %
	SysCpuUsage uint16 `json:"sysCpuUsage"` // 系统cpu使用率 (v/100).toFixed(2) %
}
