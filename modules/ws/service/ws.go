package service

import (
	"fmt"
	"sync"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
)

type WS struct {
	wsClientMap sync.Map // ws客户端 [clientId: client]
}

func NewWS() *WS {
	return &WS{}
}

// ClientAdd 客户端添加
func (s *WS) ClientAdd(wsConn *ws.ServerConn) {
	s.wsClientMap.Store(wsConn.ClientId(), wsConn)
}

// ClientRemove 客户端移除
func (s *WS) ClientRemove(wsConn *ws.ServerConn) {
	v, ok := s.wsClientMap.Load(wsConn.ClientId())
	if !ok {
		return
	}
	conn := v.(*ws.ServerConn)
	s.wsClientMap.Delete(conn.ClientId())
}

// ClientSend 给已知客户端发消息
// messageType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
func (s *WS) ClientSend(clientID string, messageType int, data map[string]any) error {
	v, ok := s.wsClientMap.Load(clientID)
	if !ok {
		return fmt.Errorf("no fount client ID: %s", clientID)
	}

	conn := v.(*ws.ServerConn)
	if len(conn.SendChan) > 90 {
		s.ClientRemove(conn)
		return fmt.Errorf("msg chan over 90 will close client ID: %s", clientID)
	}
	conn.SendRespJSON(messageType, clientID, resp.CODE_SUCCESS, resp.MSG_SUCCESS, data)
	return nil
}
