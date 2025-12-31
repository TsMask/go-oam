package pull

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/modules/pull/controller"
)

// SetupRouteConfig 网元配置路由注册
func SetupRouteConfig(router gin.IRouter) {
	config := controller.NewConfigController()
	configGroup := router.Group("/pull/config")
	{
		configGroup.GET("", config.Info)
		configGroup.PUT("", config.Edit)
		configGroup.POST("", config.Add)
		configGroup.DELETE("", config.Remove)
	}
}

// SetupRouteOMC 网管路由注册
func SetupRouteOMC(router gin.IRouter) {
	omc := controller.NewOMCController()
	omcGroup := router.Group("/pull/omc")
	{
		omcGroup.GET("/link", omc.LinkGet)
		omcGroup.PUT("/link", omc.LinkSet)
	}
}
