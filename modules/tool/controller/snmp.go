package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
)

// NewSNMPController 实例化控制层 SNMPController 结构体
func NewSNMPController() *SNMPController {
	return &SNMPController{
		srv: service.NewSNMPService(),
	}
}

// SNMP
//
// PATH /tool/snmp
type SNMPController struct {
	srv *service.SNMP // SNMP 命令交互工具服务
}

// SNMP 命令执行
//
// POST /command
//
//	@Tags			tool/snmp
//	@Summary		SNMP run
//	@Router			/tool/snmp/command [post]
func (s *SNMPController) Command(c *gin.Context) {
	var body struct {
		Oid      string `json:"oid" binding:"required"`                            // OID
		OperType string `json:"operType" binding:"required,oneof=GET GETNEXT SET"` // 操作类型
		Value    any    `json:"value"`                                             // 值
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	oamCallback := reqctx.OAMCallback(c)
	output := s.srv.Command(oamCallback, body.Oid, body.OperType, body.Value)
	c.JSON(200, resp.OkData(output))
}

// SNMP 终端会话
//
// GET /session
//
//	@Tags			tool/snmp
//	@Summary		(ws://) SNMP endpoint session
//	@Router			/tool/snmp/session [get]
func (s *SNMPController) Session(c *gin.Context) {
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
