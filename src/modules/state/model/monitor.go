package model

// MonitorInfo 机器资源信息
type MonitorInfo struct {
	MonitorBase    MonitorBase      `json:"base"`    // 监控_基本信息
	MonitorIO      []MonitorIO      `json:"io"`      // 监控_磁盘IO
	MonitorNetwork []MonitorNetwork `json:"network"` // 监控_网络IO
}

// MonitorBase 监控_基本信息
type MonitorBase struct {
	CreateTime int64   `json:"createTime"` // 创建时间
	CPU        float64 `json:"cpu"`        // cpu使用率
	LoadUsage  float64 `json:"loadUsage"`  // cpu平均使用率
	CPULoad1   float64 `json:"cpuLoad1"`   // cpu使用1分钟
	CPULoad5   float64 `json:"cpuLoad5"`   // cpu使用5分钟
	CPULoad15  float64 `json:"cpuLoad15"`  // cpu使用15分钟
	Memory     float64 `json:"memory"`     // 内存使用率
}

// MonitorIO 监控_磁盘IO
type MonitorIO struct {
	CreateTime int64  `json:"createTime"` // 创建时间
	Name       string `json:"name"`       // 磁盘名
	Read       uint64 `json:"read"`       // 读取 Bytes
	Write      uint64 `json:"write"`      // 写入 Bytes
	Count      uint64 `json:"count"`      // 次数
	Time       uint64 `json:"time"`       // 耗时
}

// MonitorNetwork 监控_网络IO
type MonitorNetwork struct {
	CreateTime int64  `json:"createTime"` // 创建时间
	Name       string `json:"name"`       // 网卡名
	Up         uint64 `json:"up"`         // 上行 bytes
	Down       uint64 `json:"down"`       // 下行 bytes
}
