package route

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/middleware"
)

// Engine 初始HTTP路由引擎
func Engine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.Use(gin.Recovery())
	// 注册中间件
	app.Use(middleware.ErrorCatch())
	// 路由未找到时
	app.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"code": 404,
			"msg":  fmt.Sprintf("Not Found %s %s", c.Request.Method, c.Request.RequestURI),
		})
	})
	// 禁止控制台日志输出的颜色
	gin.DisableConsoleColor()
	app.ForwardedByClientIP = true
	return app
}
