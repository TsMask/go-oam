package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/src/callback"
	"github.com/tsmask/go-oam/src/framework/ws"
	wsModel "github.com/tsmask/go-oam/src/modules/ws/model"
	wsService "github.com/tsmask/go-oam/src/modules/ws/service"
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
func (s Telnet) Session(conn *ws.ServerConn, msg []byte) {
	var reqMsg wsModel.WSRequest
	if err := json.Unmarshal(msg, &reqMsg); err != nil {
		wsService.SendErr(conn, "", "message format json error")
		return
	}

	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		wsService.SendErr(conn, "", "message requestId is required")
		return
	}

	switch reqMsg.Type {
	case "close":
		conn.Close()
		return
	case "ping", "PING":
		conn.Pong()
		wsService.SendOK(conn, reqMsg.RequestID, "PONG")
		return
	case "telnet":
		// Telnet会话消息接收写入会话
		if command := fmt.Sprint(reqMsg.Data); command != "" {
			output := callback.Telent(command)
			wsService.SendOK(conn, reqMsg.RequestID, output)
		}
	default:
		wsService.SendErr(conn, reqMsg.RequestID, fmt.Sprintf("message type %s not supported", reqMsg.Type))
		return
	}
}
