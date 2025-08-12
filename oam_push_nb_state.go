package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"

	"github.com/gin-gonic/gin"
)

const (
	NB_STATE_ON  = "ON"  // 基站状态-开
	NB_STATE_OFF = "OFF" // 基站状态-关
)

type NBState = model.NBState

// NBStateHistoryList 基站状态历史列表
// 需要先调用 NBStateHistoryClearTimer 才会开启记录
func NBStateHistoryList() []NBState {
	return service.NBStateHistoryList()
}

// NBStateHistoryClearTimer 基站状态历史定时清除 每小时重新记录，保留一小时
func NBStateHistoryClearTimer() {
	service.NBStateHistoryClearTimer()
}

// NBStateReceiveRoute 基站状态接收路由装载
// 接收端实现
func NBStateReceiveRoute(router gin.IRouter, onReceive func(NBState) error) {
	router.POST(service.NB_STATE_PUSH_URI, func(c *gin.Context) {
		var body NBState
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

// NBStatePushURL 基站状态推送 自定义URL地址接收
func NBStatePushURL(url string, nbState *NBState) error {
	return service.NBStatePushURL(url, nbState)
}

// NBStatePush 基站状态推送
// 默认URL地址：NB_STATE_PUSH_URI
func NBStatePush(nbState *NBState) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.NB_STATE_PUSH_URI)
	nbState.NeUid = omcInfo.NeUID
	return service.NBStatePushURL(url, nbState)
}
