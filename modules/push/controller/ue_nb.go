package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/generate"
	"github.com/tsmask/go-oam/framework/utils/parse"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

// NewUENBController 创建 UENB 控制器
func NewUENBController(srv *service.UENB) *UENBController {
	if srv == nil {
		srv = service.NewUENB()
	}
	return &UENBController{srv: srv}
}

// 终端接入基站
//
// PATH /ue/nb
type UENBController struct {
	srv *service.UENB
}

// 终端接入基站历史记录
//
// GET /history
//
//	@Tags			UENB
//	@Summary		UENB History List
//	@Router			/ue/nb/history [get]
func (s UENBController) History(c *gin.Context) {
	n := parse.Number(c.Query("n"))
	data := s.srv.HistoryList(int(n))
	c.JSON(200, resp.OkData(data))
}

// 终端接入基站发送测试
//
// GET /test
//
//	@Tags			UENB
//	@Summary		UENB Push Test
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
		NeUid:  query.NeUID,                            // 网元唯一标识
		NBId:   fmt.Sprint(generate.Number(2)),         // 基站ID
		CellId: "1",                                    // 小区ID
		TAC:    "4388",                                 // TAC
		IMSI:   fmt.Sprintf("%d", generate.Number(15)), // IMSI
		Result: model.UENB_RESULT_AUTH_SUCCESS,         // 结果值
		Type:   model.UENB_TYPE_AUTH,                   // 终端接入基站类型
	}
	err := s.srv.PushURL(query.Url, &uenb, 0)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
