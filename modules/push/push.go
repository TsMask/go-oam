package push

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/modules/push/controller"
	"github.com/tsmask/go-oam/modules/push/service"
)

// SetupRouteAlarm 告警路由注册
func SetupRouteAlarm(router gin.IRouter) {
	alarm := controller.NewAlarmController()
	alarmGroup := router.Group("/push/alarm")
	{
		alarmGroup.GET("/history", alarm.History)
		alarmGroup.GET("/test", alarm.Test)
	}
}

// SetupRouteKPI KPI 指标路由注册
func SetupRouteKPI(router gin.IRouter, neUid string, granularity time.Duration) {
	kpi := controller.NewKPIController(service.NewKPI(neUid, granularity))
	kpiGroup := router.Group("/push/kpi")
	{
		kpiGroup.GET("/history", kpi.History)
		kpiGroup.GET("/test", kpi.Test)
	}
}

// SetupRouteNBState 基站状态路由注册
func SetupRouteNBState(router gin.IRouter) {
	nbState := controller.NewNBStateController()
	nbStateGroup := router.Group("/push/nb/state")
	{
		nbStateGroup.GET("/history", nbState.History)
		nbStateGroup.GET("/test", nbState.Test)
	}
}

// SetupRouteUENB 终端接入基站路由注册
func SetupRouteUENB(router gin.IRouter) {
	uenb := controller.NewUENBController()
	uenbGroup := router.Group("/push/ue/nb")
	{
		uenbGroup.GET("/history", uenb.History)
		uenbGroup.GET("/test", uenb.Test)
	}
}

// SetupRouteCDR 话单路由注册
func SetupRouteCDR(router gin.IRouter) {
	cdr := controller.NewCDRController()
	cdrGroup := router.Group("/push/cdr")
	{
		cdrGroup.GET("/history", cdr.History)
		cdrGroup.GET("/test", cdr.Test)
	}
}

// SetupRouteCommon 通用路由注册
func SetupRouteCommon(router gin.IRouter) {
	common := controller.NewCommonController()
	commonGroup := router.Group("/push/common")
	{
		commonGroup.GET("/history", common.History)
		commonGroup.GET("/test", common.Test)
	}
}
