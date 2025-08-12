package pull

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/modules/pull/controller"
)

// 模块路由注册
func SetupRoute(router gin.IRouter) {
	// 网管路由
	omc := controller.NewOMC
	omcGroup := router.Group("/pull/omc")
	{
		omcGroup.GET("/link", omc.LinkGet)
		omcGroup.PUT("/link", omc.LinkSet)
	}
}
