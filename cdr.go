package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/push/model"
	"github.com/tsmask/go-oam/src/modules/push/service"

	"github.com/gin-gonic/gin"
)

type CDR = model.CDR

// CDRHistoryList 话单历史列表
// 需要先调用 CDRHistoryClearTimer 才会开启记录
func CDRHistoryList() []CDR {
	return service.CDRHistoryList()
}

// CDRHistoryClearTimer 话单历史定时清除 每十分钟重新记录，保留十分钟
func CDRHistoryClearTimer() {
	service.CDRHistoryClearTimer()
}

// CDRReceiveRoute 话单接收路由装载
// 接收端实现
func CDRReceiveRoute(router gin.IRouter, onReceive func(CDR) error) {
	router.POST(service.CDR_PUSH_URI, func(c *gin.Context) {
		var body CDR
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

// CDRPushURL 话单推送 自定义URL地址接收
func CDRPushURL(url string, CDR *CDR) error {
	return service.CDRPushURL(url, CDR)
}

// CDRPush 话单推送
// 默认URL地址：CDR_PUSH_URI
//
// protocol 协议 http(s)
//
// host 服务地址 如：192.168.5.58:33020
func CDRPush(protocol, host string, CDR *CDR) error {
	url := fmt.Sprintf("%s://%s%s", protocol, host, service.CDR_PUSH_URI)
	return service.CDRPushURL(url, CDR)
}
