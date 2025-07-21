package controller

import (
	"time"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// 实例化控制层 TimestampController 结构体
var NewTimestamp = &TimestampController{}

// 服务器时间
//
// PATH /time
type TimestampController struct{}

// Handler 路由
//
// GET /
//
//	@Tags			common
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Summary		Server Time
//	@Description	Server Time
//	@Router			/ [get]
func (s TimestampController) Handler(c *gin.Context) {
	now := time.Now()
	// 获取当前时间戳
	timestamp := now.UnixMilli()
	// 获取时区
	timezone := now.Format("-0700")
	// 获取时区名称
	timezoneName := now.Format("MST")
	// 获取 RFC3339 格式的时间
	rfc3339 := now.Format(time.RFC3339)
	// 获取程序运行时间
	runTime := time.Since(config.RunTime()).Abs().Seconds()
	c.JSON(200, resp.OkData(map[string]any{
		"timestamp":    timestamp,
		"timezone":     timezone,
		"timezoneName": timezoneName,
		"rfc3339":      rfc3339,
		"runTime":      int64(runTime),
	}))
}
