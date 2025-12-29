package service

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"github.com/tsmask/go-oam/modules/ws/processor"
)

// Commont 通用
//
// messageType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
func ReceiveCommon(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	var respData any
	var err error
	switch req.Type {
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type [%s] not supported", req.Type), nil)
		return
	case "ps":
		respData, err = processor.GetProcessData(req.Data)
	case "net":
		respData, err = processor.GetNetConnections(req.Data)
	case "file:upload":
		respData, err = processor.FileUpload(messageType, req.Data)
	case "file:chunk:upload":
		respData, err = processor.FileChunkUpload(messageType, req.Data)
	case "file:download":
		respData, err = processor.FileDownload(messageType, req.Data)
	}

	if err != nil {
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, err.Error(), nil)
		return
	}
	conn.SendRespJSON(messageType, req.Uuid, resp.CODE_SUCCESS, resp.MSG_SUCCESS, respData)

}
