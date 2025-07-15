package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/push/model"
	"github.com/tsmask/go-oam/src/modules/push/service"
)

// 实例化控制层 NBStateController 结构体
var NewNBState = &NBStateController{}

// 基站状态
//
// PATH /push/nb/state
type NBStateController struct{}

// 基站状态历史记录
//
// GET /history
//
//	@Tags			NBState
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		NBState Server Information
//	@Description	NBState Server Information
//	@Router			/push/nb/state/history [get]
func (s NBStateController) History(c *gin.Context) {
	data := service.NBStateHistoryList()
	c.JSON(200, resp.OkData(data))
}

// 基站状态发送测试
//
// GET /test
//
//	@Tags			NBState
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		NBState Server Information
//	@Description	NBState Server Information
//	@Router			/push/nb/state/test [get]
func (s NBStateController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	nbState := model.NBState{
		NeUid:      query.NeUID,            // 网元唯一标识
		Address:    "192.168.101.112",      // 基站地址
		DeviceName: "TestNB",               // 基站设备名称
		State:      model.NB_STATE_OFF,     // 基站状态 ON/OFF
		StateTime:  time.Now().UnixMilli(), // 基站状态时间
		Name:       "TestName",             // 基站名称 网元标记
		Position:   "TestPosition",         // 基站位置 网元标记
	}
	err := service.NBStatePushURL(query.Url, &nbState)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
