package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

// 实例化控制层 CDRController 结构体
var NewCDR = &CDRController{}

// 话单
//
// PATH /cdr
type CDRController struct{}

// 话单历史记录
//
// GET /history
//
//	@Tags			CDR
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		CDR Server Information
//	@Description	CDR Server Information
//	@Router			/cdr/history [get]
func (s CDRController) History(c *gin.Context) {
	data := service.CDRHistoryList()
	c.JSON(200, resp.OkData(data))
}

// 话单发送测试
//
// GET /test
//
//	@Tags			CDR
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		CDR Server Information
//	@Description	CDR Server Information
//	@Router			/cdr/test [get]
func (s CDRController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	CDR := model.CDR{
		NeUid: query.NeUID, //网元唯一标识
		Data: map[string]any{
			"seqNumber":    true,
			"callDuration": 76,
			"recordType":   "MOC",
			"cause":        200,
			"releaseTime":  1749697806,
		},
	}
	err := service.CDRPushURL(query.Url, &CDR)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
