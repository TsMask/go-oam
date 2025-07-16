package route

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/config"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/middleware"
	"github.com/tsmask/go-oam/src/framework/route/middleware/security"
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
	routeArr := config.Get("route")
	if routeArr == nil {
		return fmt.Errorf("route config not found")
	}
	for _, v := range routeArr.([]any) {
		item := v.(map[string]any)
		address := fmt.Sprint(item["addr"])
		schema := fmt.Sprint(item["schema"])
		if schema == "https" && schema != "<nil>" {
			certFile := fmt.Sprint(item["cert"])
			keyFile := fmt.Sprint(item["key"])
			// 启动HTTPS服务
			wg.Add(1)
			go func(addr string, certFile string, keyFile string) {
				defer wg.Done()
				for i := range 10 {
					if err := router.RunTLS(addr, certFile, keyFile); err != nil {
						logger.Errorf("route run tls err:%v", err)
						time.Sleep(10 * time.Second) // 重试间隔时间
						logger.Warnf("trying to restart HTTPS server on %s (Attempt %d)", address, i)
					}
				}
			}(address, certFile, keyFile)
		} else {
			// 启动HTTP服务
			wg.Add(1)
			go func(address string) {
				defer wg.Done()
				for i := range 10 {
					if err := router.Run(address); err != nil {
						logger.Errorf("route run err:%v", err)
						time.Sleep(10 * time.Second) // 重试间隔时间
						logger.Warnf("trying to restart HTTP server on %s (Attempt %d)", address, i)
					}
				}
			}(address)
		}
	}
	wg.Wait()
	logger.Warnf("route http server stop")
	return nil
}
