package model

const (
	NB_STATE_ON  = "ON"  // 基站状态-开
	NB_STATE_OFF = "OFF" // 基站状态-关
)

// NBState 基站状态
type NBState struct {
	NeUid      string `json:"neUid" binding:"required"`       // 网元唯一标识
	RecordTime int64  `json:"recordTime" binding:"required"`  // 记录时间 时间戳毫秒，Push时自动填充
	Address    string `json:"address"  binding:"required"`    // 基站地址
	DeviceName string `json:"deviceName"  binding:"required"` // 基站设备名称
	State      string `json:"state"  binding:"required"`      // 基站状态 ON/OFF
	StateTime  int64  `json:"stateTime"  binding:"required"`  // 基站状态时间 时间戳毫秒
	Name       string `json:"name"  binding:"required"`       // 基站名称 网元标记
	Position   string `json:"position"  binding:"required"`   // 基站位置 网元标记
}
