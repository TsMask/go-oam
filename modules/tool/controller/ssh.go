package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
)

// NewSSHController 实例化控制层 SSHController 结构体
func NewSSHController() *SSHController {
	return &SSHController{
		srv: service.NewSSHService(),
	}
}

// SSH
//
// PATH /tool/ssh
type SSHController struct {
	srv *service.SSH // SSH  终端命令交互工具服务
}

// SSH 命令执行
//
// POST /command
//
//	@Tags			tool/SSH
//	@Summary		SSH run
//	@Router			/tool/ssh/command [post]
func (s *SSHController) Command(c *gin.Context) {
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
//	@Summary		(ws://) SSH endpoint session
//	@Router			/tool/ssh/session [get]
func (s *SSHController) Session(c *gin.Context) {
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

	//  连接会话
	clientSession, err := cmd.NewClientSession(query.Cols, query.Rows)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	defer clientSession.Close()

	wsConn := ws.ServerConn{}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	wsConn.SetAnyConn(clientSession)
	go wsConn.WriteListen(nil)
	go wsConn.ReadListen(nil, s.srv.Session)

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
				wsConn.SendTextJSON(fmt.Sprintf("ssh_%d", ms.UnixMilli()), resp.CODE_SUCCESS, string(outputByte), nil)
			}
		case <-wsConn.CloseSignal(): // 等待停止信号
			wsConn.Close()
			return
		}
	}
}
