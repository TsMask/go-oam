package internal

// Record 通用数据
type Record struct {
	CoreUID    string `json:"core_uid"`    // 核心网ID
	NeUID      string `json:"ne_uid"`      // 网元ID
	RecordTime int64  `json:"record_time"` // 记录时间UTC0毫秒
	RecordType string `json:"record_type"` // 记录类型
	RecordData any    `json:"record_data"` // 记录数据
}
