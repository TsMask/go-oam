package service

import (
	"sync"

	"github.com/tsmask/go-oam/src/framework/ws"
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
