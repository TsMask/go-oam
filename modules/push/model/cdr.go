package model

// CDR 话单信息对象
type CDR struct {
	NeUid      string `json:"neUid" binding:"required"`      // 网元唯一标识
	RecordTime int64  `json:"recordTime" binding:"required"` // 记录时间 时间戳毫秒，Push时自动填充
	Data       any    `json:"data"  binding:"required"`      // 话单信息
}
