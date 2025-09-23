package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/parse"
	"github.com/tsmask/go-oam/modules/push/service"
)

// 实例化控制层 KPI Controller 结构体
var NewKPI = &KPIController{}

// 指标
//
// PATH /kpi
type KPIController struct{}

// 指标历史记录
//
// GET /history
//
//	@Tags			KPI
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		KPI Server Information
//	@Description	KPI Server Information
//	@Router			/kpi/history [get]
func (s KPIController) History(c *gin.Context) {
	n := parse.Number(c.Query("n"))
	data := service.KPIHistoryList(int(n))
	c.JSON(200, resp.OkData(data))
}

// 指标发送测试
//
// GET /test
//
//	@Tags			KPI
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		KPI Server Information
//	@Description	KPI Server Information
//	@Router			/kpi/test [get]
func (s KPIController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	err := service.KPISend(query.Url, query.NeUID, 1, map[string]float64{
		"Test.01": 10,
		"Test.02": float64(time.Now().Second()),
	})
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
