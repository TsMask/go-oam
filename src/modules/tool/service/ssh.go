package service

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/tsmask/go-oam/src/framework/cmd"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	wsModel "github.com/tsmask/go-oam/src/modules/ws/model"
)

// 实例化服务层 SSH 结构体
var NewSSH = &SSH{}

// SSH 终端命令交互工具 服务层处理
type SSH struct{}

// Session 终端交互会话-业务处理
func (s SSH) Session(client *wsModel.WSClient, reqMsg wsModel.WSRequest) {
	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		msg := "message requestId is required"
		logger.Infof("ws SSH UID %s err: %s", client.BindUid, msg)
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
	case "ssh":
		command := reqMsg.Data.(string)
		sshClientSession := client.ChildConn.(*cmd.LocalClientSession)
		_, err = sshClientSession.Write(command)
	case "ctrl-c":
		// 模拟按下 Ctrl+C
		localClientSession := client.ChildConn.(*cmd.LocalClientSession)
		_, err = localClientSession.Write("\u0003\n")
	case "resize":
		// 会话窗口重置
		msgByte, _ := json.Marshal(reqMsg.Data)
		var data struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		err = json.Unmarshal(msgByte, &data)
		if err == nil {
			sshClientSession := client.ChildConn.(*cmd.LocalClientSession)
			sshClientSession.WindowChange(data.Rows, data.Cols)
		}
	default:
		err = fmt.Errorf("message type %s not supported", reqMsg.Type)
	}

	if err != nil {
		logger.Warnf("ws SSH UID %s err: %s", client.BindUid, err.Error())
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
