package controller

import (
	"github.com/tsmask/go-oam/src/framework/config"
	"github.com/tsmask/go-oam/src/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// 实例化控制层 IndexController 结构体
var NewIndex = &IndexController{}

// 路由主页
//
// PATH /
type IndexController struct{}

// 根路由
//
// GET /
//
//	@Tags			common
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Summary		Root Route
//	@Description	Root Route
//	@Router			/ [get]
func (s *IndexController) Handler(c *gin.Context) {
	c.JSON(200, resp.OkData(map[string]any{
		"type":       config.Get("ne.type"),
		"version":    config.Get("ne.version"),
		"serialNum":  config.Get("ne.serialNum"),
		"expiryDate": config.Get("ne.expiryDate"),
		"capability": config.Get("ne.capability"),
		"validDays":  config.LicenseDaysLeft(),
	}))
}
