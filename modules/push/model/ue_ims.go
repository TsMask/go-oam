package model

const (
	UEIMS_RESULT_UNKNOWN = "0"   // 终端接入IMS认证结果-未知 0
	UEIMS_RESULT_SUCCESS = "200" // 终端接入IMS认证结果-成功 200
)

const (
	UEIMS_TYPE_REGISTER   = "InitialRegister"  // 终端接入IMS类型-初始注册
	UEIMS_TYPE_PERIODIC   = "PeriodicRegister" // 终端接入IMS类型-周期注册
	UEIMS_TYPE_UNREGISTER = "Unregister"       // 终端接入IMS类型-注销
)

// UEIMS 终端接入IMS信息对象
type UEIMS struct {
	NeUid      string `json:"neUid" binding:"required"`      // 网元唯一标识
	RecordTime int64  `json:"recordTime" binding:"required"` // 记录时间 时间戳毫秒，Push时自动填充
	IMSI       string `json:"imsi"  binding:"required"`      // IMSI
	Result     string `json:"result" binding:"required"`     // 结果值
	Type       string `json:"type" binding:"required"`       // 终端接入IMS类型
}
