package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
	wsService "github.com/tsmask/go-oam/modules/ws/service"
)

// 实例化控制层 SNMPController 结构体
var NewSNMP = &SNMPController{
	snmpService: service.NewSNMP,
}

// SNMP
//
// PATH /tool/SNMP
type SNMPController struct {
	snmpService *service.SNMP // SNMP 命令交互工具服务
}

// SNMP 命令执行
//
// POST /command
//
//	@Tags			tool/snmp
//	@Accept			json
//	@Produce		json
//	@Param			command	query		string	true	"Command"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		SNMP run
//	@Description	SNMP run
//	@Router			/tool/snmp/command [post]
func (s SNMPController) Command(c *gin.Context) {
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

	output := s.snmpService.Command(body.Oid, body.OperType, body.Value)
	c.JSON(200, resp.OkData(output))
}

// SNMP 终端会话
//
// GET /session
//
//	@Tags			tool/snmp
//	@Accept			json
//	@Produce		json
//	@Param			neUid			query		string	true	"网元唯一标识"						default(001)
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) SNMP endpoint session
//	@Description	(ws://) SNMP endpoint session
//	@Router			/tool/snmp/session [get]
func (s SNMPController) Session(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid"  binding:"required"` // 网元唯一标识
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	wsConn := ws.ServerConn{
		BindUID: query.NeUID, // 绑定唯一标识ID
	}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	go wsConn.WriteListen(1, nil)
	go wsConn.ReadListen(1, nil, s.snmpService.Session)
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
