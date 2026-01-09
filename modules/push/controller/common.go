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

// NewCommonController 创建通用推送控制器
func NewCommonController(srv *service.Common) *CommonController {
	if srv == nil {
		srv = service.NewCommon()
	}
	return &CommonController{srv: srv}
}

// 通用
//
// PATH /common
type CommonController struct {
	srv *service.Common
}

// 通用历史记录
//
// GET /history?type=x
//
//	@Tags			Common
//	@Summary		Common History List
//	@Router			/common/history [get]
func (s CommonController) History(c *gin.Context) {
	typeStr := c.Query("type")
	if typeStr == "" {
		c.JSON(200, resp.ErrMsg("type is required"))
		return
	}
	n := parse.Number(c.Query("n"))
	data := s.srv.HistoryList(typeStr, int(n))
	c.JSON(200, resp.OkData(data))
}

// 通用发送测试
//
// GET /test
//
//	@Tags			Common
//	@Summary		Common Push Test
//	@Router			/common/test [get]
func (s CommonController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
		Type  string `form:"type" binding:"required"`  // 类型
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	common := model.Common{
		NeUid: query.NeUID, //网元唯一标识
		Type:  query.Type,  //类型
		Data: map[string]any{
			"bool":  true,
			"num":   76,
			"str":   "MOC",
			"cause": generate.Code(3),
			"hax":   generate.String(12),
		},
	}
	err := s.srv.PushURL(query.Url, &common, 0)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
