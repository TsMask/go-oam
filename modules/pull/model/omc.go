package model

// OMC 网管信息对象
type OMC struct {
	Url     string `json:"url" binding:"required"`     // 网管地址 如：http://192.168.5.58:33040
	CoreUID string `json:"coreUid" binding:"required"` // 核心网唯一标识 12345678
	NeUID   string `json:"neUid" binding:"required"`   // 网元唯一标识 如：12345678
}
