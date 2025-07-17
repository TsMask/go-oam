package service

import (
	"encoding/json"
	"fmt"

	"github.com/tsmask/go-oam/src/framework/ws"
	"github.com/tsmask/go-oam/src/modules/ws/model"
	"github.com/tsmask/go-oam/src/modules/ws/processor"
)

// Commont 通用
func ReceiveCommont(conn *ws.ServerConn, msg []byte) {
	var reqMsg model.WSRequest
	if err := json.Unmarshal(msg, &reqMsg); err != nil {
		SendErr(conn, "", "message format json error")
		return
	}

	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		SendErr(conn, "", "message requestId is required")
		return
	}

	switch reqMsg.Type {
	case "close":
		conn.Close()
		return
	case "ping", "PING":
		conn.Pong()
		SendOK(conn, reqMsg.RequestID, "PONG")
		return
	case "ps":
		data, err := processor.GetProcessData(reqMsg.Data)
		if err != nil {
			SendErr(conn, reqMsg.RequestID, err.Error())
			return
		}
		SendOK(conn, reqMsg.RequestID, data)
	case "net":
		data, err := processor.GetNetConnections(reqMsg.Data)
		if err != nil {
			SendErr(conn, reqMsg.RequestID, err.Error())
			return
		}
		SendOK(conn, reqMsg.RequestID, data)
	default:
		SendErr(conn, reqMsg.RequestID, fmt.Sprintf("message type %s not supported", reqMsg.Type))
		return
	}
}
