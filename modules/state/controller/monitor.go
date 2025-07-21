package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/state/service"
)

// 实例化控制层 MonitorController 结构体
var NewMonitor = &MonitorController{}

// 机器资源
//
// PATH /monitor
type MonitorController struct{}

// 机器资源信息
//
// GET /
//
//	@Tags			state
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Monitor Server Information
//	@Description	Monitor Server Information
//	@Router			/state/monitor [get]
func (s MonitorController) Handler(c *gin.Context) {
	var query struct {
		Type     string `form:"type" binding:"required,oneof=all load io network"`  // 数据类型all/load/io/network
		Duration int    `form:"duration" binding:"required,oneof=5 10 15 20 25 30"` // 采集周期5 10 15 20 25 30
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	duration := time.Duration(query.Duration) * time.Second
	monitorService := service.NewMonitor
	switch query.Type {
	case "load":
		data := monitorService.LoadCPUMem(duration)
		c.JSON(200, resp.OkData(data))
		return
	case "io":
		data := monitorService.LoadDiskIO(duration)
		c.JSON(200, resp.OkData(data))
		return
	case "network":
		data := monitorService.LoadNetIO(duration)
		c.JSON(200, resp.OkData(data))
		return
	case "all":
		data := make(map[string]any, 0)
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			data["load"] = monitorService.LoadCPUMem(duration)
		}()
		go func() {
			defer wg.Done()
			data["io"] = monitorService.LoadDiskIO(duration)
		}()
		go func() {
			defer wg.Done()
			data["network"] = monitorService.LoadNetIO(duration)
		}()
		wg.Wait()
		c.JSON(200, resp.OkData(data))
		return
	}
}
