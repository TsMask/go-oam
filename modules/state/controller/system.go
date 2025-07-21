package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/state/service"
)

// 实例化控制层 SystemController 结构体
var NewSystem = &SystemController{}

// 系统状态
//
// PATH /system
type SystemController struct{}

// 服务器信息
//
// GET /
//
//	@Tags			state
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		System Server Information
//	@Description	System Server Information
//	@Router			/state/system [get]
func (s SystemController) Handler(c *gin.Context) {
	systemService := service.NewSystem
	data := map[string]any{
		"cpu":     systemService.CPUInfo(),
		"memory":  systemService.MemoryInfo(),
		"network": systemService.NetworkInfo(),
		"time":    systemService.TimeInfo(),
		"system":  systemService.Info(),
		"disk":    systemService.DiskInfo(),
	}
	c.JSON(200, resp.OkData(data))
}
