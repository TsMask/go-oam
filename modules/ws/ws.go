package ws

import (
	"github.com/tsmask/go-oam/modules/ws/controller"

	"github.com/gin-gonic/gin"
)

// SetupRoute 模块路由注册
func SetupRoute(router gin.IRouter) {
	ws := controller.NewWSController()
	// ws 路由
	wsGroup := router.Group("/ws")
	{
		wsGroup.GET("", ws.WS)
		wsGroup.GET("/test", ws.Test)
	}
}
