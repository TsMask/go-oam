package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/tool/service"
	wsService "github.com/tsmask/go-oam/src/modules/ws/service"
)

// 实例化控制层 SNMPController 结构体
var NewSNMP = &SNMPController{
	snmpService: service.NewSNMP,
	wsService:   wsService.NewWS,
}

// SNMP
//
// PATH /tool/SNMP
type SNMPController struct {
	snmpService *service.SNMP // SNMP 命令交互工具服务
	wsService   *wsService.WS // WebSocket 服务
}

// SNMP 命令执行
//
// POST /command
//
//	@Tags			tool/SNMP
//	@Accept			json
//	@Produce		json
//	@Param			command	query		string	true	"Command"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		SNMP run
//	@Description	SNMP run
//	@Router			/tool/SNMP/command [post]
func (s SNMPController) Command(c *gin.Context) {
	var body struct {
		Command string `form:"command" binding:"required"` // 命令
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	output := s.snmpService.Command(body.Command)
	c.JSON(200, resp.OkData(output))
}

// SNMP 终端会话
//
// GET /session
//
//	@Tags			tool/SNMP
//	@Accept			json
//	@Produce		json
//	@Param			neUid			query		string	true	"网元唯一标识"						default(001)
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) SNMP endpoint session
//	@Description	(ws://) SNMP endpoint session
//	@Router			/tool/SNMP/session [get]
func (s SNMPController) Session(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid"  binding:"required"` // 网元唯一标识
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 将 HTTP 连接升级为 WebSocket 连接
	wsConn := s.wsService.UpgraderWs(c.Writer, c.Request)
	if wsConn == nil {
		return
	}
	defer wsConn.Close()

	wsClient := s.wsService.ClientCreate(query.NeUID, nil, wsConn, nil)
	go s.wsService.ClientWriteListen(wsClient)
	go s.wsService.ClientReadListen(wsClient, s.snmpService.Session)

	// 等待停止信号
	for value := range wsClient.StopChan {
		s.wsService.ClientClose(wsClient.ID)
		logger.Infof("ws Stop Client UID %s %s", wsClient.BindUid, value)
		return
	}
}
