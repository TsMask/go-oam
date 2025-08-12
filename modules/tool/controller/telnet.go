package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
	wsService "github.com/tsmask/go-oam/modules/ws/service"
)

// 实例化控制层 TelnetController 结构体
var NewTelnet = &TelnetController{
	telnetService: service.NewTelnet,
}

// Telnet
//
// PATH /tool/telnet
type TelnetController struct {
	telnetService *service.Telnet // Telnet 命令交互工具服务
}

// Telnet 命令执行
//
// POST /command
//
//	@Tags			tool/telnet
//	@Accept			json
//	@Produce		json
//	@Param			command	query		string	true	"Command"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Telnet run
//	@Description	Telnet run
//	@Router			/tool/telnet/command [post]
func (s TelnetController) Command(c *gin.Context) {
	var body struct {
		Command string `form:"command" binding:"required"` // 命令
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	output := s.telnetService.Command(body.Command)
	c.JSON(200, resp.OkData(output))
}

// Telnet 终端会话
//
// GET /session
//
//	@Tags			tool/telnet
//	@Accept			json
//	@Produce		json
//	@Param			bindUid			query		string	true	"绑定唯一标识"						default(001)
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) Telnet endpoint session
//	@Description	(ws://) Telnet endpoint session
//	@Router			/tool/telnet/session [get]
func (s TelnetController) Session(c *gin.Context) {
	var query struct {
		BindUID string `form:"bindUid"  binding:"required"` // 绑定唯一标识
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	wsConn := ws.ServerConn{
		BindUID: query.BindUID, // 绑定唯一标识ID
	}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	go wsConn.WriteListen(1, nil)
	go wsConn.ReadListen(1, nil, s.telnetService.Session)
	// 发客户端id确认是否连接
	wsService.SendOK(&wsConn, "", map[string]string{
		"clientId": wsConn.ClientId(),
	})

	// 等待停止信号
	for range wsConn.StopChan {
		wsConn.Close()
		return
	}
}
