package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
)

// NewOMCController 实例化控制层 OMCController 结构体
func NewOMCController() *OMCController {
	return &OMCController{}
}

// 网管
//
// PATH /omc
type OMCController struct{}

// 网管连接信息
//
// GET /link
//
//	@Tags			omc
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		OMC Server Information
//	@Description	OMC Server Information
//	@Router			/omc/link [get]
func (s *OMCController) LinkGet(c *gin.Context) {
	oamCfg := reqctx.OAMConfig(c)
	var data config.OMCConfig
	oamCfg.View(func(c *config.Config) {
		data = c.OMC
	})
	c.JSON(200, resp.OkData(data))
}

// 网管连接设置
//
// PUT /link
//
//	@Tags			omc
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		OMC Server Link
//	@Description	OMC Server Link
//	@Router			/omc/link [put]
func (s *OMCController) LinkSet(c *gin.Context) {
	var body config.OMCConfig
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}
	oamCfg := reqctx.OAMConfig(c)
	oamCfg.Update(func(c *config.Config) {
		c.OMC = body
	})
	c.JSON(200, resp.Ok(nil))
}
