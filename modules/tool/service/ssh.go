package service

import (
	"encoding/json"
	"fmt"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
)

// 实例化服务层 SSH 结构体
var NewSSH = &SSH{}

// SSH 终端命令交互工具 服务层处理
type SSH struct{}

// Session 终端交互会话-业务处理
func (s SSH) Session(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	switch req.Type {
	case "ssh":
		command := fmt.Sprint(req.Data)
		if command != "" && command != "<nil>" {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			if _, err := localClientSession.Write(command); err != nil {
				conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, err.Error(), nil)
			}
		}
	case "ctrl-c":
		// 模拟按下 Ctrl+C
		localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
		if _, err := localClientSession.Write("\u0003\n"); err != nil {
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, err.Error(), nil)
		}
	case "resize":
		// 会话窗口重置
		var data struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		msgByte, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(msgByte, &data); err == nil {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			localClientSession.WindowChange(data.Cols, data.Rows)
		}
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type %s not supported", req.Type), nil)
		return
	}
}
