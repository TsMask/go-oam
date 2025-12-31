package service

import (
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"github.com/tsmask/go-oam/modules/callback"
)

func NewTelnetService() *Telnet {
	return &Telnet{}
}

// Telnet 命令交互工具 服务层处理
type Telnet struct {
}

// Command 执行单次命令 "help"
func (s *Telnet) Command(handler callback.CallbackHandler, cmdStr string) string {
	if handler == nil {
		return "telnet unrealized"
	}
	output := handler.Telnet(cmdStr)
	return strings.TrimSpace(output)
}

// Session 接收终端交互业务处理
func (s *Telnet) Session(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	switch req.Type {
	case "telnet":
		// Telnet会话消息接收写入会话
		if command := fmt.Sprint(req.Data); command != "" {
			handler := conn.GetAnyConn().(callback.CallbackHandler)
			if handler == nil {
				conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, "callback unrealized", nil)
				return
			}
			output := handler.Telnet(command)
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_SUCCESS, output, nil)
			return
		}
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type [%s] not supported", req.Type), nil)
		return
	}
}
