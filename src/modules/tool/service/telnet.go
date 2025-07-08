package service

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tsmask/go-oam/src/callback"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	wsModel "github.com/tsmask/go-oam/src/modules/ws/model"
)

// 实例化服务层 Telnet 结构体
var NewTelnet = &Telnet{}

// Telnet 命令交互工具 服务层处理
type Telnet struct{}

// Command 执行单次命令 "help"
func (s Telnet) Command(cmdStr string) string {
	output := callback.Telent(cmdStr)
	return strings.TrimSpace(output)
}

// Telnet 接收终端交互业务处理
func (s Telnet) Session(client *wsModel.WSClient, reqMsg wsModel.WSRequest) {
	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		msg := "message requestId is required"
		logger.Infof("ws Telnet UID %s err: %s", client.BindUid, msg)
		msgByte, _ := json.Marshal(resp.ErrMsg(msg))
		client.MsgChan <- msgByte
		return
	}

	var resByte []byte
	var err error

	switch reqMsg.Type {
	case "close":
		// 主动关闭
		resultByte, _ := json.Marshal(resp.OkMsg("user initiated closure"))
		client.MsgChan <- resultByte
		// 等待1s后关闭连接
		time.Sleep(1 * time.Second)
		client.StopChan <- struct{}{}
		return
	case "telnet":
		// Telnet会话消息接收写入会话
		command := fmt.Sprint(reqMsg.Data)
		output := callback.Telent(command)
		msgByte, _ := json.Marshal(resp.OkData(output))
		client.MsgChan <- msgByte
	default:
		err = fmt.Errorf("message type %s not supported", reqMsg.Type)
	}

	if err != nil {
		logger.Warnf("ws Telnet UID %s err: %s", client.BindUid, err.Error())
		msgByte, _ := json.Marshal(resp.ErrMsg(err.Error()))
		client.MsgChan <- msgByte
		if err == io.EOF {
			// 等待1s后关闭连接
			time.Sleep(1 * time.Second)
			client.StopChan <- struct{}{}
		}
		return
	}
	if len(resByte) > 0 {
		client.MsgChan <- resByte
	}
}
