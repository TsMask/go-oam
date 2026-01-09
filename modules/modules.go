package modules

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route"
)

// NewRouter 创建路由引擎
func NewRouter() *gin.Engine {
	return route.Engine()
}

// RunServer 启动服务
func RunServer(cfg *config.Config, router *gin.Engine) error {
	var routeConfigs []config.RouteConfig
	cfg.View(func(c *config.Config) {
		routeConfigs = c.Route
	})
	if len(routeConfigs) == 0 {
		return fmt.Errorf("route config not found")
	}

	var wg sync.WaitGroup
	for _, v := range routeConfigs {
		address := v.Addr
		schema := v.Schema
		if schema == "https" {
			certFile := v.Cert
			keyFile := v.Key
			// 启动HTTPS服务
			wg.Add(1)
			go func(address string, certFile string, keyFile string) {
				defer wg.Done()
				for i := range 10 {
					if err := router.RunTLS(address, certFile, keyFile); err != nil {
						log.Printf("[OAM] route run https on %s (Attempt %d) error, %s\n", address, i, err.Error())
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
						log.Printf("[OAM] route run http on %s (Attempt %d) error, %s\n", address, i, err.Error())
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
