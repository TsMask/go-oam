package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"

	"github.com/gin-gonic/gin"
)

type Common = model.Common

// CommonPush 通用推送
// 默认URL地址：COMMON_PUSH_URI
func CommonPush(common *Common) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.COMMON_PUSH_URI)
	common.NeUid = omcInfo.NeUID
	return service.CommonPushURL(url, common)
}

// CommonPushURL 通用推送 自定义URL地址接收
func CommonPushURL(url string, common *Common) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	return service.CommonPushURL(url, common)
}

// CommonHistoryList 通用历史列表
// n 为返回的最大记录数，n<0返回空列表
func CommonHistoryList(typeStr string, n int) []Common {
	return service.CommonHistoryList(typeStr, n)
}

// CommonHistorySetSize 设置通用历史列表数量
// 当设置的大小小于当前历史记录数时，会自动清理旧记录
// 默认值 4096
func CommonHistorySetSize(size int) {
	service.CommonHistorySetSize(size)
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
