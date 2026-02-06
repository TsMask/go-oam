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

// NewUEIMSController 创建 UEIMS 控制器
func NewUEIMSController(srv *service.UEIMS) *UEIMSController {
	if srv == nil {
		srv = service.NewUEIMS()
	}
	return &UEIMSController{srv: srv}
}

// 终端接入IMS
//
// PATH /ue/ims
type UEIMSController struct {
	srv *service.UEIMS
}

// 终端接入IMS历史记录
//
// GET /history
//
//	@Tags			UEIMS
//	@Summary		UEIMS History List
//	@Router			/ue/ims/history [get]
func (s UEIMSController) History(c *gin.Context) {
	n := parse.Number(c.Query("n"))
	data := s.srv.HistoryList(int(n))
	c.JSON(200, resp.OkData(data))
}

// 终端接入IMS发送测试
//
// GET /test
//
//	@Tags			UEIMS
//	@Summary		UEIMS Push Test
//	@Router			/ue/ims/test [get]
func (s UEIMSController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	ueims := model.UEIMS{
		NeUid:  query.NeUID,                            // 网元唯一标识
		IMSI:   fmt.Sprintf("%d", generate.Number(15)), // IMSI
		Result: model.UEIMS_RESULT_SUCCESS,             // 结果值
		Type:   model.UEIMS_TYPE_REGISTER,              // 终端接入IMS类型
	}
	err := s.srv.PushURL(query.Url, &ueims, 0)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
