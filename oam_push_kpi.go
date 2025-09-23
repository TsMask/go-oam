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

// KPIPush KPI推送
// 默认URL地址：KPI_PUSH_URI
func KPIPush(kpi *KPI) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.KPI_PUSH_URI)
	kpi.NeUid = omcInfo.NeUID
	return service.KPISend(url, kpi.NeUid, kpi.Granularity, kpi.Data)
}

// KPIPushURL KPI推送 自定义URL地址接收
// 默认URL地址：KPI_PUSH_URI
func KPIPushURL(url string, kpi *KPI) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	return service.KPISend(url, kpi.NeUid, kpi.Granularity, kpi.Data)
}

// KPIHistoryList KPI历史列表
// n 为返回的最大记录数，n<0返回空列表
func KPIHistoryList(n int) []KPI {
	return service.KPIHistoryList(n)
}

// KPIHistorySetSize 设置KPI历史列表数量
// 当设置的大小小于当前历史记录数时，会自动清理旧记录
// 默认值 4096
func KPIHistorySetSize(size int) {
	service.KPIHistorySetSize(size)
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

// KPITimerStart KPI开启定时上报
// 默认URL地址：KPI_PUSH_URI
func KPITimerStart(duration time.Duration) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.KPI_PUSH_URI)
	KPITimerStartURL(url, omcInfo.NeUID, duration)
	return nil
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

// KPIKeyGet 对Key原子获取
func KPIKeyGet(key string) float64 {
	return kpiTimer.KeyGet(key)
}

// KPIKeySet 对Key原子设置
func KPIKeySet(key string, v float64) {
	kpiTimer.KeySet(key, v)
}

// KPIKeyInc 对Key原子累加1
func KPIKeyInc(key string) {
	kpiTimer.KeyInc(key)
}

// KPIKeyDec 对Key原子累减1
func KPIKeyDec(key string) {
	kpiTimer.KeyDec(key)
}

// KPIKeyDel 对Key原子删除
func KPIKeyDel(key string) {
	kpiTimer.KeyDel(key)
}
