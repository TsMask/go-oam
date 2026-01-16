package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/modules/ws"
)

// 网元内已有gin的情况，兼容现有的oam_manager
func main() {
	r := gin.Default()
	// oanGroup := r.Group("/oam")

	// 加入OAM相关接口模块
	o := oam.New(
		oam.WithNEConfig(config.NEConfig{
			Type:       "NE",
			Version:    "1.0",
			SerialNum:  "1234567890",
			ExpiryDate: "2025-12-31",
			NbNumber:   10,
			UeNumber:   100,
		}),
	)
	o.SetupCallback(new(oamCallback))
	o.SetupRoute(ws.SetupRoute)
	o.RouteEngine(r)

	r.Run(":33030")
}

// oamCallback 回调功能
type oamCallback struct{}

// Standby implements callback.CallbackHandler.
func (o *oamCallback) Standby() bool {
	return false
}

// Redis implements callback.CallbackHandler.
func (o *oamCallback) Redis(args ...any) (any, error) {
	// *redis.Client
	// return client.Do(ctx, args...).Result()
	return nil, nil
}

// Telnet implements callback.CallbackHandler.
func (o *oamCallback) Telnet(command string) string {
	return "Telnet implements"
}

// SNMP implements callback.CallbackHandler.
func (o *oamCallback) SNMP(oid, operType string, value any) any {
	return "SNMP implements"
}

// Config implements callback.CallbackHandler.
func (o *oamCallback) Config(action, paramName, loc string, paramValue any) error {
	return fmt.Errorf("config => %s > %s > %s > %v", action, paramName, loc, paramValue)
}
