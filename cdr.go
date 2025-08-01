package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"

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
func CDRPushURL(url string, cdr *CDR) error {
	return service.CDRPushURL(url, cdr)
}

// CDRPush 话单推送
// 默认URL地址：CDR_PUSH_URI
func CDRPush(cdr *CDR) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.CDR_PUSH_URI)
	cdr.NeUid = omcInfo.NeUID
	return service.CDRPushURL(url, cdr)
}
