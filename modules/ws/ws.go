package ws

import (
	"github.com/tsmask/go-oam/modules/ws/controller"

	"github.com/gin-gonic/gin"
)

// 模块路由注册
func SetupRoute(router gin.IRouter) {
	// WebSocket 协议
	ws := controller.NewWSController
    wsGroup := router.Group("/ws")
    {
        wsGroup.GET("", ws.WS) // ws
        wsGroup.GET("/test", ws.Test)
        wsGroup.GET("/binary", ws.WSBinary)
    }
}
