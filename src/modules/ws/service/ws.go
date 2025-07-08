package service

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/framework/utils/generate"
	"github.com/tsmask/go-oam/src/modules/ws/model"
)

var (
	wsClients sync.Map // ws客户端 [clientId: client]
	wsUsers   sync.Map // ws用户对应的多个客户端id [uid:clientIds]
	wsGroup   sync.Map // ws组对应的多个客户端id [groupId:clientIds]
)

// NewWS 实例化服务层 WS 结构体
var NewWS = &WS{}

// WS WebSocket通信 服务层处理
type WS struct{}

// UpgraderWs http升级ws请求
func (s *WS) UpgraderWs(w http.ResponseWriter, r *http.Request) *websocket.Conn {
	wsUpgrader := websocket.Upgrader{
		Subprotocols: []string{"oam-ws"},
		// 设置消息发送缓冲区大小（byte），如果这个值设置得太小，可能会导致服务端在发送大型消息时遇到问题
		WriteBufferSize: 4096,
		// 消息包启用压缩
		EnableCompression: true,
		// ws握手超时时间
		HandshakeTimeout: 5 * time.Second,
		// ws握手过程中允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("ws Upgrade err: %s", err.Error())
	}
	return conn
}

// ClientCreate 客户端新建
//
// uid 唯一标识ID
// groupIDs 订阅组
// conn ws连接实例
// childConn 子连接实例
func (s *WS) ClientCreate(uid string, groupIDs []string, conn *websocket.Conn, childConn any) *model.WSClient {
	// clientID也可以用其他方式生成，只要能保证在所有服务端中都能保证唯一即可
	clientID := generate.Code(16)

	wsClient := &model.WSClient{
		ID:            clientID,
		Conn:          conn,
		LastHeartbeat: time.Now().UnixMilli(),
		BindUid:       uid,
		SubGroup:      groupIDs,
		MsgChan:       make(chan []byte, 100),
		StopChan:      make(chan struct{}, 1), //  卡死循环标记
		ChildConn:     childConn,
	}

	// 存入客户端
	wsClients.Store(clientID, wsClient)

	// 存入用户持有客户端
	if uid != "" {
		if v, ok := wsUsers.Load(uid); ok {
			uidClientIds := v.(*[]string)
			*uidClientIds = append(*uidClientIds, clientID)
		} else {
			wsUsers.Store(uid, &[]string{clientID})
		}
	}

	// 存入用户订阅组
	if uid != "" && len(groupIDs) > 0 {
		for _, groupID := range groupIDs {
			if v, ok := wsGroup.Load(groupID); ok {
				groupClientIds := v.(*[]string)
				*groupClientIds = append(*groupClientIds, clientID)
			} else {
				wsGroup.Store(groupID, &[]string{clientID})
			}
		}
	}

	return wsClient
}

// ClientClose 客户端关闭
func (s *WS) ClientClose(clientID string) {
	v, ok := wsClients.Load(clientID)
	if !ok {
		return
	}

	client := v.(*model.WSClient)
	defer func() {
		client.MsgChan <- []byte("ws:close")
		client.StopChan <- struct{}{}
		client.Conn.Close()
		wsClients.Delete(clientID)
	}()

	// 客户端断线时自动踢出Uid绑定列表
	if client.BindUid != "" {
		if v, ok := wsUsers.Load(client.BindUid); ok {
			uidClientIds := v.(*[]string)
			if len(*uidClientIds) > 0 {
				tempClientIds := make([]string, 0, len(*uidClientIds))
				for _, v := range *uidClientIds {
					if v != client.ID {
						tempClientIds = append(tempClientIds, v)
					}
				}
				*uidClientIds = tempClientIds
			}
		}
	}

	// 客户端断线时自动踢出已加入的组
	if len(client.SubGroup) > 0 {
		for _, groupID := range client.SubGroup {
			v, ok := wsGroup.Load(groupID)
			if !ok {
				continue
			}
			groupClientIds := v.(*[]string)
			if len(*groupClientIds) > 0 {
				tempClientIds := make([]string, 0, len(*groupClientIds))
				for _, v := range *groupClientIds {
					if v != client.ID {
						tempClientIds = append(tempClientIds, v)
					}
				}
				*groupClientIds = tempClientIds
			}
		}
	}
}

// ClientReadListen 客户端读取消息监听
// receiveFn 接收函数进行消息处理
func (s *WS) ClientReadListen(wsClient *model.WSClient, receiveFn func(*model.WSClient, model.WSRequest)) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("ws ReadMessage Panic Error: %v", err)
		}
	}()
	for {
		// 读取消息
		messageType, msg, err := wsClient.Conn.ReadMessage()
		if err != nil {
			logger.Warnf("ws ReadMessage UID %s err: %s", wsClient.BindUid, err.Error())
			s.ClientClose(wsClient.ID)
			return
		}
		// fmt.Println(messageType, string(msg))

		// 文本 只处理文本json
		if messageType == websocket.TextMessage {
			var reqMsg model.WSRequest
			if err := json.Unmarshal(msg, &reqMsg); err != nil {
				msgByte, _ := json.Marshal(resp.ErrMsg("message format json error"))
				wsClient.MsgChan <- msgByte
				continue
			}
			// 接收器处理
			go receiveFn(wsClient, reqMsg)
		}
	}
}

// ClientWriteListen 客户端写入消息监听
// wsClient.MsgChan <- msgByte 写入消息
func (s *WS) ClientWriteListen(wsClient *model.WSClient) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("ws WriteMessage Panic Error: %v", err)
		}
	}()
	// 发客户端id确认是否连接
	msgByte, _ := json.Marshal(resp.OkData(map[string]string{
		"clientId": wsClient.ID,
	}))
	wsClient.MsgChan <- msgByte
	// 消息发送监听
	for msg := range wsClient.MsgChan {
		// PONG句柄
		if string(msg) == "ws:pong" {
			wsClient.LastHeartbeat = time.Now().UnixMilli()
			wsClient.Conn.WriteMessage(websocket.PongMessage, []byte{})
			continue
		}
		// 关闭句柄
		if string(msg) == "ws:close" {
			wsClient.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 发送消息
		err := wsClient.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			logger.Warnf("ws WriteMessage UID %s err: %s", wsClient.BindUid, err.Error())
			s.ClientClose(wsClient.ID)
			return
		}
	}
}
