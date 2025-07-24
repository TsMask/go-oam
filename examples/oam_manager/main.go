package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam"
)

// 网元内已有gin的情况，兼容现有的oam_manager
func main() {
	r := gin.Default()
	// oanGroup := r.Group("/oam")

	// 加入OAM相关接口模块
	o := oam.New(&oam.Opts{
		License: &oam.License{
			NeType:     "NE",
			Version:    "1.0",
			SerialNum:  "1234567890",
			ExpiryDate: "2025-12-31",
			Capability: 100,
		},
	})
	o.SetupCallback(new(oamCallback))
	// if err := o.RouteExpose(oanGroup); err != nil {
	if err := o.RouteExpose(r); err != nil {
		fmt.Printf("oam run fail: %s\n", err.Error())
	}

	r.Run(":33030")
}

// oamCallback 回调功能
type oamCallback struct{}

// Standby implements callback.CallbackHandler.
func (o *oamCallback) Standby() bool {
	return false
}

// Redis implements callback.CallbackHandler.
func (o *oamCallback) Redis() any {
	// *redis.Client
	return nil
}

// Telent implements callback.CallbackHandler.
func (o *oamCallback) Telent(command string) string {
	return "Telent implements"
}

// SNMP implements callback.CallbackHandler.
func (o *oamCallback) SNMP(oid, operType string, value any) any {
	return "SNMP implements"
}
