package route

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route/middleware"
	"github.com/tsmask/go-oam/framework/route/middleware/security"
)

// Engine 初始HTTP路由引擎
func Engine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.Use(gin.Recovery())
	// 注册中间件
	app.Use(
		middleware.ErrorCatch(),
		middleware.Cors(middleware.CorsDefaultOpt),
		security.Security(security.SecurityDefaultOpt),
	)
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

func Run(router *gin.Engine) error {
	// 开启HTTP服务
	var wg sync.WaitGroup
	routeArr, ok := config.Get("route").([]any)
	if routeArr == nil || !ok {
		return fmt.Errorf("route config not found")
	}
	for _, v := range routeArr {
		item, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("route config info error")
		}
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
						log.Printf("[OAM] route run https error, %s\n", err.Error())
						log.Printf("[OAM] trying to restart HTTPS server on %s (Attempt %d)\n", address, i)
						// 等待指数退避的时间
						backoffTime := time.Duration(1<<i) * time.Second // 2^i 秒
						time.Sleep(backoffTime)
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
						log.Printf("[OAM] route run http error, %s\n", err.Error())
						log.Printf("[OAM] trying to restart HTTP server on %s (Attempt %d)\n", address, i)
						// 等待指数退避的时间
						backoffTime := time.Duration(1<<i) * time.Second // 2^i 秒
						time.Sleep(backoffTime)
					}
				}
			}(address)
		}
	}
	wg.Wait()
	return nil
}
