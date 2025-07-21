package service

import (
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
)

// SendErr 发送失败消息
func SendErr(conn *ws.ServerConn, requestId, str string) {
	if requestId == "" {
		conn.SendJSON(resp.ErrMsg(str))
		return
	}
	conn.SendJSON(resp.Err(map[string]any{
		"requestId": requestId,
		"msg":       str,
	}))
}

// SendOK 发送OK消息
func SendOK(conn *ws.ServerConn, requestId string, data any) {
	if requestId == "" {
		conn.SendJSON(resp.OkData(data))
		return
	}
	conn.SendJSON(resp.Ok(map[string]any{
		"requestId": requestId,
		"data":      data,
	}))
}
