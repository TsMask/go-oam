package oam

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

type KPI = model.KPI

var kpiTimer *service.KPI

// KPITimerStart KPI开启定时上报
// 默认URL地址：/kpi/receive
//
// protocol 协议 http(s)
//
// host 服务地址 如：192.168.5.58:33020
func KPITimerStart(protocol, host, neUid string, duration time.Duration) {
	url := fmt.Sprintf("%s://%s%s", protocol, host, service.KPI_PUSH_URI)
	KPITimerStartURL(url, neUid, duration)
}

// KPITimerStartURL KPI开启定时上报
//
// url 自定义URL地址接收
//
// duration 周期 60 * time.Second
func KPITimerStartURL(url string, neUid string, duration time.Duration) {
	kpiTimer = &service.KPI{
		NeUid:       neUid,
		Granularity: duration,
	}
	kpiTimer.KPITimerStart(url)
}

// KPITimerStop KPI停止定时上报
func KPITimerStop() {
	kpiTimer.KPITimerStop()
}

// KPIKeySet 对Key原子设置
func KPIKeySet(key string, v float64) {
	kpiTimer.KeySet(key, v)
}

// KPIKeyInc 对Key原子累加
func KPIKeyInc(key string) {
	kpiTimer.KeyInc(key)
}

// KPIKeyDec 对Key原子累减
func KPIKeyDec(key string) {
	kpiTimer.KeyDec(key)
}

// KPIKeyGet 对Key原子获取
func KPIKeyGet(key string) float64 {
	return kpiTimer.KeyGet(key)
}

// KPIHistoryList KPI历史列表
func KPIHistoryList() []KPI {
	return service.KPIHistoryList()
}

// KPIReceiveRoute 告警接收路由装载
// 接收端实现
func KPIReceiveRoute(router gin.IRouter, onReceive func(KPI) error) {
	router.POST(service.KPI_PUSH_URI, func(c *gin.Context) {
		var body KPI
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
