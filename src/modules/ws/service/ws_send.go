package service

import (
	"encoding/json"
	"fmt"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/ws/model"
)

// 订阅组指定编号为支持服务器向客户端主动推送数据
const (
	// 组号-其他
	GROUP_OTHER = "0"
)

// 实例化服务层 WSSend 结构体
var NewWSSend = &WSSend{}

// WSSend WebSocket消息发送处理 服务层处理
type WSSend struct{}

// ByClientID 给已知客户端发消息
func (s *WSSend) ByClientID(clientID string, data any) error {
	v, ok := wsClients.Load(clientID)
	if !ok {
		return fmt.Errorf("no fount client ID: %s", clientID)
	}

	dataByte, err := json.Marshal(resp.OkData(data))
	if err != nil {
		return err
	}

	client := v.(*model.WSClient)
	if len(client.MsgChan) > 90 {
		NewWS.ClientClose(client.ID)
		return fmt.Errorf("msg chan over 90 will close client ID: %s", clientID)
	}
	client.MsgChan <- dataByte
	return nil
}

// ByGroupID 给订阅组的客户端发送消息
func (s *WSSend) ByGroupID(groupID string, data any) error {
	clientIds, ok := wsGroup.Load(groupID)
	if !ok {
		return fmt.Errorf("no fount Group ID: %s", groupID)
	}

	// 检查组内是否有客户端
	ids := clientIds.(*[]string)
	if len(*ids) == 0 {
		return fmt.Errorf("no members in the group")
	}

	// 遍历给客户端发消息
	for _, clientId := range *ids {
		err := s.ByClientID(clientId, map[string]any{
			"groupId": groupID,
			"data":    data,
		})
		if err != nil {
			continue
		}
	}

	return nil
}
