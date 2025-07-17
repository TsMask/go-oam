package service

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/framework/ws"
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

// ByClientID 给已知客户端发消息
func ByClientID(clientID string, data any) error {
	v, ok := wsClientMap.Load(clientID)
	if !ok {
		return fmt.Errorf("no fount client ID: %s", clientID)
	}

	conn := v.(*ws.ServerConn)
	if len(conn.SendChan) > 90 {
		ClientRemove(conn)
		return fmt.Errorf("msg chan over 90 will close client ID: %s", clientID)
	}
	SendOK(conn, clientID, data)
	return nil
}
