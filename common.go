package oam

import (
	"fmt"
	"time"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"

	"github.com/gin-gonic/gin"
)

type Common = model.Common

// CommonHistoryList 通用历史列表
// 需要先调用 CommonHistoryClearTimer 才会开启记录
func CommonHistoryList(typeStr string) []Common {
	return service.CommonHistoryList(typeStr)
}

// CommonHistoryClearTimer 通用历史定时清除
func CommonHistoryClearTimer(typeStr string, d time.Duration) {
	service.CommonHistoryClearTimer(typeStr, d)
}

// CommonReceiveRoute 通用接收路由装载
// 接收端实现
func CommonReceiveRoute(router gin.IRouter, onReceive func(Common) error) {
	router.POST(service.COMMON_PUSH_URI, func(c *gin.Context) {
		var body Common
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

// CommonPushURL 通用推送 自定义URL地址接收
func CommonPushURL(url string, common *Common) error {
	return service.CommonPushURL(url, common)
}

// CommonPush 通用推送
// 默认URL地址：COMMON_PUSH_URI
//
// protocol 协议 http(s)
//
// host 服务地址 如：192.168.5.58:33020
func CommonPush(protocol, host string, common *Common) error {
	url := fmt.Sprintf("%s://%s%s", protocol, host, service.COMMON_PUSH_URI)
	return service.CommonPushURL(url, common)
}
