package state

import (
	"github.com/tsmask/go-oam/modules/state/controller"

	"github.com/gin-gonic/gin"
)

// SetupRoute 模块路由注册
func SetupRoute(router gin.IRouter) {
	state := controller.NewStateController()
	system := controller.NewSystemController()
	monitor := controller.NewMonitorController()

	// 网元状态
	router.GET("/state/standby", state.Standby)
	router.GET("/state/ne", state.NE)

	// 系统状态
	router.GET("/state/system", system.Handler)

	// 机器资源状态
	router.GET("/state/monitor", monitor.Handler)
}
