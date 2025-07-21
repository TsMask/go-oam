package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/generate"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

// 实例化控制层 CommonController 结构体
var NewCommon = &CommonController{}

// 通用
//
// PATH /common
type CommonController struct{}

// 通用历史记录
//
// GET /history?type=x
//
//	@Tags			Common
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Common Server Information
//	@Description	Common Server Information
//	@Router			/common/history [get]
func (s CommonController) History(c *gin.Context) {
	typeStr := c.Query("type")
	if typeStr == "" {
		c.JSON(200, resp.ErrMsg("type is required"))
		return
	}
	data := service.CommonHistoryList(typeStr)
	c.JSON(200, resp.OkData(data))
}

// 通用发送测试
//
// GET /test
//
//	@Tags			Common
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Common Server Information
//	@Description	Common Server Information
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

	Common := model.Common{
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
	err := service.CommonPushURL(query.Url, &Common)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
