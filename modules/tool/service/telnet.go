package service

import (
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"github.com/tsmask/go-oam/modules/callback"
)

// 实例化服务层 Telnet 结构体
var NewTelnet = &Telnet{}

// Telnet 命令交互工具 服务层处理
type Telnet struct{}

// Command 执行单次命令 "help"
func (s Telnet) Command(cmdStr string) string {
	output := callback.Telent(cmdStr)
	return strings.TrimSpace(output)
}

// Telnet 接收终端交互业务处理
func (s Telnet) Session(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	switch req.Type {
	case "telnet":
		// Telnet会话消息接收写入会话
		if command := fmt.Sprint(req.Data); command != "" {
			output := callback.Telent(command)
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_SUCCESS, output, nil)
			return
		}
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type [%s] not supported", req.Type), nil)
		return
	}
}
