package service

import (
	"encoding/json"
	"fmt"

	"github.com/tsmask/go-oam/src/framework/cmd"
	"github.com/tsmask/go-oam/src/framework/ws"
	wsModel "github.com/tsmask/go-oam/src/modules/ws/model"
	wsService "github.com/tsmask/go-oam/src/modules/ws/service"
)

// 实例化服务层 SSH 结构体
var NewSSH = &SSH{}

// SSH 终端命令交互工具 服务层处理
type SSH struct{}

// Session 终端交互会话-业务处理
func (s SSH) Session(conn *ws.ServerConn, msg []byte) {
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
	case "ssh":
		if command := fmt.Sprint(reqMsg.Data); command != "" && command != "<nil>" {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			if _, err := localClientSession.Write(command); err != nil {
				wsService.SendErr(conn, reqMsg.RequestID, err.Error())
			}
		}
	case "ctrl-c":
		// 模拟按下 Ctrl+C
		localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
		if _, err := localClientSession.Write("\u0003\n"); err != nil {
			wsService.SendErr(conn, reqMsg.RequestID, err.Error())
		}
	case "resize":
		// 会话窗口重置
		var data struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		msgByte, _ := json.Marshal(reqMsg.Data)
		if err := json.Unmarshal(msgByte, &data); err == nil {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			localClientSession.WindowChange(data.Cols, data.Rows)
		}
	default:
		wsService.SendErr(conn, reqMsg.RequestID, fmt.Sprintf("message type %s not supported", reqMsg.Type))
		return
	}
}
