package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/ws/model"
	"github.com/tsmask/go-oam/src/modules/ws/processor"
)

// 实例化服务层 WSReceive 结构体
var NewWSReceive = &WSReceive{}

// WSReceive WebSocket消息接收处理 服务层处理
type WSReceive struct{}

// close 关闭服务连接
func (s *WSReceive) close(client *model.WSClient) {
	// 主动关闭
	resultByte, _ := json.Marshal(resp.OkMsg("user initiated closure"))
	client.MsgChan <- resultByte
	// 等待1s后关闭连接
	time.Sleep(1 * time.Second)
	NewWS.ClientClose(client.ID)
}

// Commont 通用-业务处理
func (s *WSReceive) Commont(client *model.WSClient, reqMsg model.WSRequest) {
	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		msg := "message requestId is required"
		logger.Infof("ws Commont UID %s err: %s", client.BindUid, msg)
		msgByte, _ := json.Marshal(resp.ErrMsg(msg))
		client.MsgChan <- msgByte
		return
	}

	var resByte []byte
	var err error

	switch reqMsg.Type {
	case "close":
		s.close(client)
		return
	case "ping", "PING":
		resByte, _ := json.Marshal(resp.Ok(map[string]any{
			"requestId": reqMsg.RequestID,
			"data":      "PONG",
		}))
		client.MsgChan <- resByte
		client.MsgChan <- []byte("ws:pong")
		return
	case "ps":
		resByte, err = processor.GetProcessData(reqMsg.RequestID, reqMsg.Data)
	case "net":
		resByte, err = processor.GetNetConnections(reqMsg.RequestID, reqMsg.Data)
	default:
		err = fmt.Errorf("message type %s not supported", reqMsg.Type)
	}

	if err != nil {
		logger.Warnf("ws Commont UID %s err: %s", client.BindUid, err.Error())
		msgByte, _ := json.Marshal(resp.ErrMsg(err.Error()))
		client.MsgChan <- msgByte
		return
	}
	if len(resByte) > 0 {
		client.MsgChan <- resByte
	}
}
