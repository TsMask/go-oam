package model

// Common 通用信息对象
type Common struct {
	NeUid      string `json:"neUid" binding:"required"`      // 网元唯一标识
	RecordTime int64  `json:"recordTime" binding:"required"` // 记录时间 时间戳毫秒，Push时自动填充
	Type       string `json:"type" binding:"required"`       // 消息类型
	Data       any    `json:"data"  binding:"required"`      // 通用信息
}
