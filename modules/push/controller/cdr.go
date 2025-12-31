package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/parse"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

// NewCDRController 创建话单控制器
func NewCDRController() *CDRController {
	return &CDRController{srv: service.NewCDR()}
}

// 话单
//
// PATH /cdr
type CDRController struct {
	srv *service.CDR
}

// 话单历史记录
//
// GET /history
//
//	@Tags			CDR
//	@Summary		CDR History List
//	@Router			/cdr/history [get]
func (s CDRController) History(c *gin.Context) {
	n := parse.Number(c.Query("n"))
	data := s.srv.HistoryList(int(n))
	c.JSON(200, resp.OkData(data))
}

// 话单发送测试
//
// GET /test
//
//	@Tags			CDR
//	@Summary		CDR Push Test
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

	cdr := model.CDR{
		NeUid: query.NeUID, //网元唯一标识
		Data: map[string]any{
			"seqNumber":    true,
			"callDuration": 76,
			"recordType":   "MOC",
			"cause":        200,
			"releaseTime":  1749697806,
		},
	}
	err := s.srv.PushURL(query.Url, &cdr)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
