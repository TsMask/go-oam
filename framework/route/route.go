package route

import (
	"fmt"
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
			return fmt.Errorf("route config not found")
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
						fmt.Printf("[OAM] route run tls error => %v\n", err)
						time.Sleep(10 * time.Second) // 重试间隔时间
						fmt.Printf("[OAM] trying to restart HTTPS server on %s (Attempt %d)\n", address, i)
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
						fmt.Printf("[OAM] route run error => %v\n", err)
						time.Sleep(10 * time.Second) // 重试间隔时间
						fmt.Printf("[OAM] trying to restart HTTP server on %s (Attempt %d)\n", address, i)
					}
				}
			}(address)
		}
	}
	wg.Wait()
	fmt.Println("route server stop")
	return nil
}
