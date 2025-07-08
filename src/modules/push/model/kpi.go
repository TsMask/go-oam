package model

// KPI 指标信息对象
type KPI struct {
	NeUid       string             `json:"neUid" binding:"required"`       // 网元唯一标识
	RecordTime  int64              `json:"recordTime" binding:"required"`  // 记录时间 时间戳毫秒，Push时自动填充
	Granularity int64              `json:"granularity" binding:"required"` // 时间间隔 5/10/.../60/300 (秒)
	Data        map[string]float64 `json:"data"  binding:"required"`       // 指标信息
}
