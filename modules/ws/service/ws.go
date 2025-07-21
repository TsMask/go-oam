package service

import (
	"fmt"
	"sync"

	"github.com/tsmask/go-oam/framework/ws"
)

var wsClientMap sync.Map // ws客户端 [clientId: client]

// ClientAdd 客户端添加
func ClientAdd(wsConn *ws.ServerConn) {
	wsClientMap.Store(wsConn.ClientId(), wsConn)
}

// ClientRemove 客户端移除
func ClientRemove(wsConn *ws.ServerConn) {
	v, ok := wsClientMap.Load(wsConn.ClientId())
	if !ok {
		return
	}
	conn := v.(*ws.ServerConn)
	wsClientMap.Delete(conn.ClientId())
}

// ClientSend 给已知客户端发消息
func ClientSend(clientID string, data any) error {
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
