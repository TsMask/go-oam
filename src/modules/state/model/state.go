package model

// State 网元状态
type State struct {
	HostName string `json:"hostName"` // linux命令: hostname
	OsInfo   string `json:"osInfo"`   // linux命令: uname -a

	IpAddr []string `json:"ipAddr"` // 网络的ipv4和ipv6列表

	Version    string `json:"version"` // 软件版本信息: 16.1.1
	Capability int64  `json:"capability"`
	SerialNum  string `json:"serialNum"`
	ExpiryDate string `json:"expiryDate"`
	Standby    bool   `json:"standby"` // 主备

	CpuUsage  CpuUsage  `json:"cpuUsage"`
	MemUsage  MemUsage  `json:"memUsage"`
	DiskSpace DiskSpace `json:"diskSpace"`
}

// MemUsage 网元状态内存信息
type MemUsage struct {
	TotalMem    uint64 `json:"totalMem"`
	NfUsedMem   uint64 `json:"nfUsedMem"`
	SysMemUsage uint64 `json:"sysMemUsage"`
}

// DiskSpace 网元状态磁盘信息
type DiskSpace struct {
	PartitionNum  uint8           `json:"partitionNum"`
	PartitionInfo []PartitionInfo `json:"partitionInfo"`
}

// PartitionInfo 网元状态磁盘使用信息
type PartitionInfo struct {
	Device string `json:"device"` // 驱动名称
	Total  uint64 `json:"total"`  // MB
	Used   uint64 `json:"used"`   // MB
}

// CpuUsage 网元状态cpu信息
type CpuUsage struct {
	NfCpuUsage  uint16 `json:"nfCpuUsage"`
	SysCpuUsage uint16 `json:"sysCpuUsage"`
}
