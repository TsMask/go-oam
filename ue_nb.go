package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"

	"github.com/gin-gonic/gin"
)

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

type UENB = model.UENB

// UENBHistoryList 终端接入基站历史列表
// 需要先调用 UENBHistoryClearTimer 才会开启记录
func UENBHistoryList() []UENB {
	return service.UENBHistoryList()
}

// UENBHistoryClearTimer 终端接入基站历史定时清除 每小时重新记录，保留一小时
func UENBHistoryClearTimer() {
	service.UENBHistoryClearTimer()
}

// UENBReceiveRoute 终端接入基站接收路由装载
// 接收端实现
func UENBReceiveRoute(router gin.IRouter, onReceive func(UENB) error) {
	router.POST(service.UENB_PUSH_URI, func(c *gin.Context) {
		var body UENB
		if err := c.ShouldBindBodyWithJSON(&body); err != nil {
			errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
			c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
			return
		}
		if err := onReceive(body); err != nil {
			c.JSON(200, resp.ErrMsg(err.Error()))
			return
		}
		c.JSON(200, resp.Ok(nil))
	})
}

// UENBPushURL 终端接入基站推送 自定义URL地址接收
func UENBPushURL(url string, uenb *UENB) error {
	return service.UENBPushURL(url, uenb)
}

// UENBPush 终端接入基站推送
// 默认URL地址：UENB_PUSH_URI
func UENBPush(uenb *UENB) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.UENB_PUSH_URI)
	uenb.NeUid = omcInfo.NeUID
	return service.UENBPushURL(url, uenb)
}
