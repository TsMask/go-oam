package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/callback"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/state/service"
)

// 实例化控制层 StateController 结构体
var NewState = &StateController{}

// 网元状态
//
// PATH /state
type StateController struct{}

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
func (s StateController) NE(c *gin.Context) {
	data := service.NewState.Info()
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
func (s StateController) Standby(c *gin.Context) {
	c.JSON(200, resp.OkData(callback.Standby()))
}
