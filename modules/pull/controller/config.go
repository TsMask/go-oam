package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/pull/model"
)

// NewConfigController 实例化控制层 ConfigController 结构体
func NewConfigController() *ConfigController {
	return &ConfigController{}
}

// 网元配置
//
// PATH /config
type ConfigController struct{}

// 网元配置信息
//
// GET /
//
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Config Data Information
//	@Description	Config Data Information
//	@Router			/config [get]
func (s *ConfigController) Info(c *gin.Context) {
	var query model.Config
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	oamCb := reqctx.OAMCallback(c)
	if oamCb == nil {
		c.JSON(200, resp.ErrMsg("callback unrealized"))
		return
	}

	err := oamCb.Config("Read", query.ParamName, query.Loc, query.ParamValue)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(nil))
}

// 网元配置更新
//
// PUT /
//
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Config Data Edit
//	@Description	Config Data Edit
//	@Router			/config [put]
func (s *ConfigController) Edit(c *gin.Context) {
	var body model.Config
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	oamCb := reqctx.OAMCallback(c)
	if oamCb == nil {
		c.JSON(200, resp.ErrMsg("callback unrealized"))
		return
	}

	err := oamCb.Config("Update", body.ParamName, body.Loc, body.ParamValue)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(nil))
}

// 网元配置新增 array
//
// POST /
//
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Config Data Add
//	@Description	Config Data Add
//	@Router			/config [post]
func (s *ConfigController) Add(c *gin.Context) {
	var body model.Config
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}
	if body.Loc == "" {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, "loc is empty"))
		return
	}

	oamCb := reqctx.OAMCallback(c)
	if oamCb == nil {
		c.JSON(200, resp.ErrMsg("callback unrealized"))
		return
	}

	err := oamCb.Config("Create", body.ParamName, body.Loc, body.ParamValue)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(nil))
}

// 网元配置删除 array
//
// DELETE /
//
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Config Data Remove
//	@Description	Config Data Remove
//	@Router			/config [delete]
func (s *ConfigController) Remove(c *gin.Context) {
	var body model.Config
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}
	if body.Loc == "" {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, "loc is empty"))
		return
	}

	oamCb := reqctx.OAMCallback(c)
	if oamCb == nil {
		c.JSON(200, resp.ErrMsg("callback unrealized"))
		return
	}

	err := oamCb.Config("Delete", body.ParamName, body.Loc, body.ParamValue)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(nil))
}
