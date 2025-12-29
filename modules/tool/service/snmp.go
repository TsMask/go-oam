package service

import (
	"encoding/json"
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"github.com/tsmask/go-oam/modules/callback"
)

// 实例化服务层 SNMP 结构体
var NewSNMP = &SNMP{}

// SNMP 终端命令交互工具 服务层处理
type SNMP struct{}

// Command 执行单次命令 "1.3.6.1.4.1.1373.2.3.3.55.1"
func (s SNMP) Command(oid, operType string, value any) any {
	output := callback.SNMP(oid, operType, value)
	return output
}

// SNMP 接收终端交互业务处理
func (s SNMP) Session(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	switch req.Type {
	case "snmp":
		// SNMP会话消息接收写入会话
		var data struct {
			Oid      string `json:"oid" binding:"required"`
			OperType string `json:"operType" binding:"required,oneof=GET GETNEXT SET"`
			Value    any    `json:"value"`
		}
		msgByte, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(msgByte, &data); err == nil {
			output := callback.SNMP(data.Oid, data.OperType, data.Value)
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_SUCCESS, resp.MSG_SUCCESS, output)
		}
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type %s not supported", req.Type), nil)
		return
	}
}
