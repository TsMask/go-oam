package model

const (
	UENB_RESULT_AUTH_SUCCESS                            = "200" // 终端接入基站认证结果-成功 200
	UENB_RESULT_AUTH_NETWORK_FAILURE                    = "001" // 终端接入基站认证结果-网络失败 001
	UENB_RESULT_AUTH_INTERFACE_FAILURE                  = "002" // 终端接入基站认证结果-接口失败 002
	UENB_RESULT_AUTH_MAC_FAILURE                        = "003" // 终端接入基站认证结果-MAC失败 003
	UENB_RESULT_AUTH_SYNC_FAILURE                       = "004" // 终端接入基站认证结果-同步失败 004
	UENB_RESULT_AUTH_NON_5G_AUTHENTICATION_NOT_ACCEPTED = "005" // 终端接入基站认证结果-不接受非5G认证 005
	UENB_RESULT_AUTH_RESPONSE_FAILURE                   = "006" // 终端接入基站认证结果-响应失败 006
	UENB_RESULT_AUTH_UNKNOWN                            = "007" // 终端接入基站认证结果-未知 007
	UENB_RESULT_CM_CONNECTED                            = "1"   // 终端接入基站连接结果-连接 1
	UENB_RESULT_CM_IDLE                                 = "2"   // 终端接入基站连接结果-空闲 2
	UENB_RESULT_CM_INACTIVE                             = "3"   // 终端接入基站连接结果-不活动 3
)

const (
	UENB_TYPE_AUTH   = "Auth"   // 终端接入基站类型-认证
	UENB_TYPE_DETACH = "Detach" // 终端接入基站类型-注销
	UENB_TYPE_CM     = "CM"     // 终端接入基站类型-连接
)

// UENB 终端接入基站信息对象
type UENB struct {
	NeUid      string `json:"neUid" binding:"required"`      // 网元唯一标识
	RecordTime int64  `json:"recordTime" binding:"required"` // 记录时间 时间戳毫秒，Push时自动填充
	NBId       string `json:"nbId"  binding:"required"`      // 基站ID
	CellId     string `json:"cellId"  binding:"required"`    // 小区ID
	TAC        string `json:"tac"  binding:"required"`       // TAC
	IMSI       string `json:"imsi"  binding:"required"`      // IMSI
	Result     string `json:"result" binding:"required"`     // 结果值
	Type       string `json:"type" binding:"required"`       // 终端接入基站类型
}
