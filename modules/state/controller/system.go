package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/state/service"
)

// NewSystemController 实例化控制层 SystemController 结构体
func NewSystemController() *SystemController {
	return &SystemController{srv: service.NewSystemService()}
}

// 系统状态
//
// PATH /system
type SystemController struct {
	srv *service.System
}

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
func (s *SystemController) Handler(c *gin.Context) {
	oamCfg := reqctx.OAMConfig(c)
	data := map[string]any{
		"cpu":     s.srv.CPUInfo(),
		"memory":  s.srv.MemoryInfo(),
		"network": s.srv.NetworkInfo(),
		"time":    s.srv.TimeInfo(),
		"system":  s.srv.Info(oamCfg),
		"disk":    s.srv.DiskInfo(),
	}
	c.JSON(200, resp.OkData(data))
}
