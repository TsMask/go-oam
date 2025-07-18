package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/cmd"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/framework/ws"
	"github.com/tsmask/go-oam/src/modules/tool/service"
	wsService "github.com/tsmask/go-oam/src/modules/ws/service"
)

// 实例化控制层 SSHController 结构体
var NewSSH = &SSHController{
	sshService: service.NewSSH,
}

// SSH
//
// PATH /tool/ssh
type SSHController struct {
	sshService *service.SSH // SSH  终端命令交互工具服务
}

// SSH 命令执行
//
// POST /command
//
//	@Tags			tool/SSH
//	@Accept			json
//	@Produce		json
//	@Param			command	query		string	true	"Command"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		SSH run
//	@Description	SSH run
//	@Router			/tool/ssh/command [post]
func (s SSHController) Command(c *gin.Context) {
	var body struct {
		Command string `form:"command" binding:"required"` // 命令
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	output, err := cmd.Exec(body.Command)
	data := strings.TrimSpace(output)
	if err != nil {
		c.JSON(200, resp.ErrMsg(fmt.Sprintf("%s; %s", err.Error(), data)))
		return
	}
	c.JSON(200, resp.OkData(data))
}

// SSH 终端会话
//
// GET /session
//
//	@Tags			tool/SSH
//	@Accept			json
//	@Produce		json
//	@Param			neUid			query		string	true	"网元唯一标识"						default(001)
//	@Param			cols			query		number	false	"Terminal line characters"	default(120)
//	@Param			rows			query		number	false	"Terminal display lines"	default(40)
//	@Param			access_token	query		string	true	"Authorization"
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) SSH endpoint session
//	@Description	(ws://) SSH endpoint session
//	@Router			/tool/ssh/session [get]
func (s SSHController) Session(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid"  binding:"required"` // 网元唯一标识
		Cols  int    `form:"cols"`                      // 终端单行字符数
		Rows  int    `form:"rows"`                      // 终端显示行数
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	//  连接会话
	clientSession, err := cmd.NewClientSession(query.Cols, query.Rows)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	defer clientSession.Close()

	wsConn := ws.ServerConn{
		BindUID: query.NeUID, // 绑定唯一标识ID
	}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	wsConn.SetAnyConn(clientSession)
	go wsConn.WriteListen(1, nil)
	go wsConn.ReadListen(1, nil, s.sshService.Session)
	// 发客户端id确认是否连接
	wsService.SendOK(&wsConn, "", map[string]string{
		"clientId": wsConn.ClientId(),
	})

	// 等待1秒，排空首次消息
	time.Sleep(1 * time.Second)
	_ = clientSession.Read()

	// 实时读取SSH消息直接输出
	msTicker := time.NewTicker(100 * time.Millisecond)
	defer msTicker.Stop()
	for {
		select {
		case ms := <-msTicker.C:
			outputByte := clientSession.Read()
			if len(outputByte) > 0 {
				wsService.SendOK(&wsConn, fmt.Sprintf("ssh_%d", ms.UnixMilli()), string(outputByte))
			}
		case <-wsConn.StopChan: // 等待停止信号
			wsConn.Close()
			return
		}
	}
}
