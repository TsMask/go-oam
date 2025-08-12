package push

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/modules/push/controller"
)

// 模块路由注册
func SetupRoute(router gin.IRouter) {
	// 告警路由
	alarm := controller.NewAlarm
	alarmGroup := router.Group("/push/alarm")
	{
		alarmGroup.GET("/history", alarm.History)
		alarmGroup.GET("/test", alarm.Test)
	}

	// KPI 指标路由
	kpi := controller.NewKPI
	kpiGroup := router.Group("/push/kpi")
	{
		kpiGroup.GET("/history", kpi.History)
		kpiGroup.GET("/test", kpi.Test)
	}

	// NBState 基站状态路由
	nbState := controller.NewNBState
	nbStateGroup := router.Group("/push/nb/state")
	{
		nbStateGroup.GET("/history", nbState.History)
		nbStateGroup.GET("/test", nbState.Test)
	}

	// UENB 终端接入基站路由
	uenb := controller.NewUENB
	uenbGroup := router.Group("/push/ue/nb")
	{
		uenbGroup.GET("/history", uenb.History)
		uenbGroup.GET("/test", uenb.Test)
	}

	// CDR 话单路由
	cdr := controller.NewCDR
	cdrGroup := router.Group("/push/cdr")
	{
		cdrGroup.GET("/history", cdr.History)
		cdrGroup.GET("/test", cdr.Test)
	}

	// 通用路由
	common := controller.NewCommon
	commonGroup := router.Group("/push/common")
	{
		commonGroup.GET("/history", common.History)
		commonGroup.GET("/test", common.Test)
	}
}
