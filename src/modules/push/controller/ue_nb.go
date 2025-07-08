package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/push/model"
	"github.com/tsmask/go-oam/src/modules/push/service"
)

// 实例化控制层 UENBController 结构体
var NewUENB = &UENBController{}

// 终端接入基站
//
// PATH /ue/nb
type UENBController struct{}

// 终端接入基站历史记录
//
// GET /history
//
//	@Tags			UENB
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		UENB Server Information
//	@Description	UENB Server Information
//	@Router			/ue/nb/history [get]
func (s UENBController) History(c *gin.Context) {
	data := service.UENBHistoryList()
	c.JSON(200, resp.OkData(data))
}

// 终端接入基站发送测试
//
// GET /test
//
//	@Tags			UENB
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		UENB Server Information
//	@Description	UENB Server Information
//	@Router			/ue/nb/test [get]
func (s UENBController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	uenb := model.UENB{
		NeUid:  query.NeUID,                 // 网元唯一标识
		NBId:   "257",                       // 基站ID
		CellId: "1",                         // 小区ID
		TAC:    "4388",                      // TAC
		IMSI:   "460991100000000",           // IMSI
		Result: model.UENBResultAuthSuccess, // 结果值
		Type:   model.UENBTypeAuth,          // 终端接入基站类型
	}
	err := service.UENBPushURL(query.Url, &uenb)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
