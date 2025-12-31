package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
)

// NewTelnetController 实例化控制层 TelnetController 结构体
func NewTelnetController() *TelnetController {
	return &TelnetController{
		srv: service.NewTelnetService(),
	}
}

// Telnet
//
// PATH /tool/telnet
type TelnetController struct {
	srv *service.Telnet // Telnet 命令交互工具服务
}

// Telnet 命令执行
//
// POST /command
//
//	@Tags			tool/telnet
//	@Summary		Telnet run
//	@Router			/tool/telnet/command [post]
func (s *TelnetController) Command(c *gin.Context) {
	var body struct {
		Command string `form:"command" binding:"required"` // 命令
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	oamCallback := reqctx.OAMCallback(c)
	output := s.srv.Command(oamCallback, body.Command)
	c.JSON(200, resp.OkData(output))
}

// Telnet 终端会话
//
// GET /session
//
//	@Tags			tool/telnet
//	@Summary		(ws://) Telnet endpoint session
//	@Router			/tool/telnet/session [get]
func (s *TelnetController) Session(c *gin.Context) {
	var query struct {
		Cols int `form:"cols"` // 终端单行字符数
		Rows int `form:"rows"` // 终端显示行数
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}
	if query.Cols == 0 {
		query.Cols = 120
	}
	if query.Rows == 0 {
		query.Rows = 40
	}

	wsConn := ws.ServerConn{}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	oamCallback := reqctx.OAMCallback(c)
	wsConn.SetAnyConn(oamCallback)
	go wsConn.WriteListen(nil)
	go wsConn.ReadListen(nil, s.srv.Session)

	// 等待停止信号
	for range wsConn.CloseSignal() {
		wsConn.Close()
		return
	}
}
