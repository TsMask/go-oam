package model

const (
	UENBResultAuthSuccess                        = "200" // 终端接入基站认证结果-成功 200
	UENBResultAuthNetworkFailure                 = "001" // 终端接入基站认证结果-网络失败 001
	UENBResultAuthInterfaceFailure               = "002" // 终端接入基站认证结果-接口失败 002
	UENBResultAuthMACFailure                     = "003" // 终端接入基站认证结果-MAC失败 003
	UENBResultAuthSyncFailure                    = "004" // 终端接入基站认证结果-同步失败 004
	UENBResultAuthNon5GAuthenticationNotAccepted = "005" // 终端接入基站认证结果-不接受非5G认证 005
	UENBResultAuthResponseFailure                = "006" // 终端接入基站认证结果-响应失败 006
	UENBResultAuthUnknown                        = "007" // 终端接入基站认证结果-未知 007
	UENBResultCMConnected                        = "1"   // 终端接入基站连接结果-连接 1
	UENBResultCMIdle                             = "2"   // 终端接入基站连接结果-空闲 2
	UENBResultCMInactive                         = "3"   // 终端接入基站连接结果-不活动 3
)

const (
	UENBTypeAuth   = "Auth"   // 终端接入基站类型-认证
	UENBTypeDetach = "Detach" // 终端接入基站类型-注销
	UENBTypeCM     = "CM"     // 终端接入基站类型-连接
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
