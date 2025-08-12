package state

import (
	"github.com/tsmask/go-oam/modules/state/controller"

	"github.com/gin-gonic/gin"
)

// 模块路由注册
func SetupRoute(router gin.IRouter) {
	// 网元状态
	state := controller.NewState
	router.GET("/state/standby", state.Standby)
	router.GET("/state/ne", state.NE)

	// 系统状态
	router.GET("/state/system", controller.NewSystem.Handler)
	// 机器资源状态
	router.GET("/state/monitor", controller.NewMonitor.Handler)
}
