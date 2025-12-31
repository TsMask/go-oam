package controller

import (
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// NewIndexController 实例化控制层 IndexController 结构体
func NewIndexController() *IndexController {
	return &IndexController{}
}

// 路由主页
//
// PATH /
type IndexController struct{}

// 根路由
//
// GET /i
//
//	@Tags			common
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Summary		Root Route
//	@Description	Root Route
//	@Router			/i [get]
func (s *IndexController) Handler(c *gin.Context) {
	var neConf config.NEConfig
	var validDays int64
	oamCfg := reqctx.OAMConfig(c)
	oamCfg.View(func(conf *config.Config) {
		neConf = conf.NE
		validDays = oamCfg.LicenseDaysLeft()
	})
	if neConf.Type == "" {
		c.JSON(200, resp.ErrMsg("ne config not found"))
		return
	}
	c.JSON(200, resp.OkData(map[string]any{
		"type":       neConf.Type,
		"version":    neConf.Version,
		"serialNum":  neConf.SerialNum,
		"expiryDate": neConf.ExpiryDate,
		"ueNumber":   neConf.UeNumber,
		"nbNumber":   neConf.NbNumber,
		"validDays":  validDays,
	}))
}
