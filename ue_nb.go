package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/push/model"
	"github.com/tsmask/go-oam/src/modules/push/service"

	"github.com/gin-gonic/gin"
)

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

type UENB = model.UENB

// UENBHistoryList 终端接入基站历史列表
// 需要先调用 AlarmHistoryClearTimer 才会开启记录
func UENBHistoryList() []UENB {
	return service.UENBHistoryList()
}

// UENBHistoryClearTimer 终端接入基站历史定时清除 每小时重新记录，保留一小时
func UENBHistoryClearTimer() {
	service.UENBHistoryClearTimer()
}

// UENBReceiveRoute 终端接入基站接收路由装载
// 接收端实现
func UENBReceiveRoute(router gin.IRouter, onReceive func(UENB)) {
	router.POST(service.UENB_PUSH_URI, func(c *gin.Context) {
		var body UENB
		if err := c.ShouldBindBodyWithJSON(&body); err != nil {
			errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
			c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
			return
		}
		onReceive(body)
		c.JSON(200, resp.Ok(nil))
	})
}

// UENBPushURL 终端接入基站推送 自定义URL地址接收
func UENBPushURL(url string, uenb *UENB) error {
	return service.UENBPushURL(url, uenb)
}

// UENBPush 终端接入基站推送
// 默认URL地址：UENB_PUSH_URI
//
// protocol 协议 http(s)
//
// host 服务地址 如：192.168.5.58:33020
func UENBPush(protocol, host string, uenb *UENB) error {
	url := fmt.Sprintf("%s://%s%s", protocol, host, service.UENB_PUSH_URI)
	return service.UENBPushURL(url, uenb)
}
