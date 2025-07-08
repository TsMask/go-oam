package route

import (
	"fmt"
	"net"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/config"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/middleware"
	"github.com/tsmask/go-oam/src/framework/route/middleware/security"
	"github.com/tsmask/go-oam/src/framework/utils/parse"
)

// Engine 初始HTTP路由引擎
func Engine(dev bool) *gin.Engine {
	var app *gin.Engine

	// 根据运行环境注册引擎
	if !dev {
		gin.SetMode(gin.ReleaseMode)
		app = gin.New()
		app.Use(gin.Recovery())
	} else {
		app = gin.Default()
	}

	// 路由未找到时
	app.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"code": 404,
			"msg":  fmt.Sprintf("Not Found %s %s", c.Request.Method, c.Request.RequestURI),
		})
	})

	// 注册中间件
	app.Use(
		middleware.ErrorCatch(),
		middleware.Cors(),
		security.Security(),
	)

	// 禁止控制台日志输出的颜色
	gin.DisableConsoleColor()
	app.ForwardedByClientIP = true
	return app
}

func Run(router *gin.Engine) error {
	// 开启HTTP服务
	var wg sync.WaitGroup
	httpArr := config.Get("route")
	if httpArr == nil {
		return fmt.Errorf("route config not found")
	}
	for _, v := range httpArr.([]any) {
		item := v.(map[string]any)
		host := fmt.Sprint(item["host"])
		port := parse.Number(item["port"])
		address := net.JoinHostPort(host, fmt.Sprint(port))
		schema := fmt.Sprint(item["schema"])
		if schema == "https" && schema != "<nil>" {
			certFile := fmt.Sprint(item["cert"])
			keyFile := fmt.Sprint(item["key"])
			// 启动HTTPS服务
			wg.Add(1)
			go func(addr string, certFile string, keyFile string) {
				defer wg.Done()
				err := router.RunTLS(addr, certFile, keyFile)
				logger.Errorf("route RunTLS err:%v", err)
			}(address, certFile, keyFile)
		} else {
			// 启动HTTP服务
			wg.Add(1)
			go func(address string) {
				defer wg.Done()
				err := router.Run(address)
				logger.Errorf("route Run err:%v", err)
			}(address)
		}
	}
	wg.Wait()
	logger.Warnf("route http server stop")
	return nil
}
