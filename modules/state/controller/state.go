package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/state/service"
)

// NewStateController 实例化控制层 StateController 结构体
func NewStateController() *StateController {
	return &StateController{srv: service.NewStateService()}
}

// 网元状态
//
// PATH /state
type StateController struct {
	srv *service.State
}

// 网元状态信息
//
// GET /ne
//
//	@Tags			state
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		State Server Information
//	@Description	State Server Information
//	@Router			/state/ne [get]
func (s *StateController) NE(c *gin.Context) {
	oamCfg := reqctx.OAMConfig(c)
	oamCallback := reqctx.OAMCallback(c)
	data := s.srv.Info(oamCfg, oamCallback)
	c.JSON(200, resp.OkData(data))
}

// 备用状态查询
//
// GET /standby
//
//	@Tags			state
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		State Server Information
//	@Description	State Server Information
//	@Router			/state/standby [get]
func (s *StateController) Standby(c *gin.Context) {
	oamCallback := reqctx.OAMCallback(c)
	data := s.srv.Standby(oamCallback)
	c.JSON(200, resp.OkData(data))
}
