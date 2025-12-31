package push

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/modules/push/controller"
	"github.com/tsmask/go-oam/modules/push/service"
)

// SetupRouteAlarm 告警路由注册
func SetupRouteAlarm(router gin.IRouter, srv *service.Alarm) error {
	if srv == nil {
		return fmt.Errorf("Alarm service is nil")
	}
	alarm := controller.NewAlarmController(srv)
	alarmGroup := router.Group("/push/alarm")
	{
		alarmGroup.GET("/history", alarm.History)
		alarmGroup.GET("/test", alarm.Test)
	}
	return nil
}

// SetupRouteKPI KPI 指标路由注册
func SetupRouteKPI(router gin.IRouter, srv *service.KPI) error {
	if srv == nil {
		return fmt.Errorf("KPI service is nil")
	}
	kpi := controller.NewKPIController(srv)
	kpiGroup := router.Group("/push/kpi")
	{
		kpiGroup.GET("/history", kpi.History)
		kpiGroup.GET("/test", kpi.Test)
	}
	return nil
}

// SetupRouteNBState 基站状态路由注册
func SetupRouteNBState(router gin.IRouter, srv *service.NBState) error {
	if srv == nil {
		return fmt.Errorf("NBState service is nil")
	}
	nbState := controller.NewNBStateController(srv)
	nbStateGroup := router.Group("/push/nb/state")
	{
		nbStateGroup.GET("/history", nbState.History)
		nbStateGroup.GET("/test", nbState.Test)
	}
	return nil
}

// SetupRouteUENB 终端接入基站路由注册
func SetupRouteUENB(router gin.IRouter, srv *service.UENB) error {
	if srv == nil {
		return fmt.Errorf("UENB service is nil")
	}
	uenb := controller.NewUENBController(srv)
	uenbGroup := router.Group("/push/ue/nb")
	{
		uenbGroup.GET("/history", uenb.History)
		uenbGroup.GET("/test", uenb.Test)
	}
	return nil
}

// SetupRouteCDR 话单路由注册
func SetupRouteCDR(router gin.IRouter, srv *service.CDR) error {
	if srv == nil {
		return fmt.Errorf("CDR service is nil")
	}
	cdr := controller.NewCDRController(srv)
	cdrGroup := router.Group("/push/cdr")
	{
		cdrGroup.GET("/history", cdr.History)
		cdrGroup.GET("/test", cdr.Test)
	}
	return nil
}

// SetupRouteCommon 通用路由注册
func SetupRouteCommon(router gin.IRouter, srv *service.Common) error {
	if srv == nil {
		return fmt.Errorf("Common service is nil")
	}
	common := controller.NewCommonController(srv)
	commonGroup := router.Group("/push/common")
	{
		commonGroup.GET("/history", common.History)
		commonGroup.GET("/test", common.Test)
	}
	return nil
}
