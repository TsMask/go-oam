package service

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tsmask/go-oam/src/framework/cmd"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/tool/model"
	wsModel "github.com/tsmask/go-oam/src/modules/ws/model"
)

// 实例化服务层 Ping 结构体
var NewPing = &Ping{}

// Ping 网络性能测试工具 服务层处理
type Ping struct{}

// Statistics ping基本信息
func (s Ping) Statistics(ping model.Ping) (map[string]any, error) {
	pinger, err := ping.NewPinger()
	if err != nil {
		return nil, err
	}
	if err = pinger.Run(); err != nil {
		return nil, err
	}
	defer pinger.Stop()
	stats := pinger.Statistics()
	return map[string]any{
		"minTime":  stats.MinRtt.Microseconds(),    // 最小时延（整数类型，可选，单位：微秒）
		"maxTime":  stats.MaxRtt.Microseconds(),    // 最大时延（整数类型，可选，单位：微秒）
		"avgTime":  stats.AvgRtt.Microseconds(),    // 平均时延（整数类型，可选，单位：微秒）
		"lossRate": int64(stats.PacketLoss),        // 丢包率（整数类型，可选，单位：%）
		"jitter":   stats.StdDevRtt.Microseconds(), // 时延抖动（整数类型，可选，单位：微秒）
	}, nil
}

// Version 查询版本信息
func (s Ping) Version() (string, error) {
	// 检查是否安装ping
	output, err := cmd.Exec("ping -V")
	if err != nil {
		return "", fmt.Errorf("ping not installed")
	}
	return strings.TrimSpace(output), err
}

// Run 接收ping终端交互业务处理
func (s Ping) Run(client *wsModel.WSClient, reqMsg wsModel.WSRequest) {
	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		msg := "message requestId is required"
		logger.Infof("ws Ping run UID %s err: %s", client.BindUid, msg)
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
	case "ping":
		// SSH会话消息接收写入会话
		var command string
		command, err = s.parseOptions(reqMsg.Data)
		if command != "" && err == nil {
			sshClientSession := client.ChildConn.(*cmd.LocalClientSession)
			_, err = sshClientSession.Write(command)
		}
	case "ctrl-c":
		// 模拟按下 Ctrl+C
		sshClientSession := client.ChildConn.(*cmd.LocalClientSession)
		_, err = sshClientSession.Write("\u0003\n")
	case "resize":
		// 会话窗口重置
		msgByte, _ := json.Marshal(reqMsg.Data)
		var data struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		err = json.Unmarshal(msgByte, &data)
		if err == nil {
			localClientSession := client.ChildConn.(*cmd.LocalClientSession)
			localClientSession.WindowChange(data.Cols, data.Rows)
		}
	default:
		err = fmt.Errorf("message type %s not supported", reqMsg.Type)
	}

	if err != nil {
		logger.Warnf("ws Ping run UID %s err: %s", client.BindUid, err.Error())
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

// parseOptions 解析拼装ping命令 ping [options] <destination>
func (s Ping) parseOptions(reqData any) (string, error) {
	msgByte, _ := json.Marshal(reqData)
	var data struct {
		Command string `json:"command"` // 命令字符串
		DesAddr string `json:"desAddr"` // dns name or ip address
		// Options
		Interval int `json:"interval"` //  seconds between sending each packet
		TTL      int `json:"ttl"`      // define time to live
		Cunt     int `json:"count"`    // <count> 次回复后停止
		Size     int `json:"size"`     // 使用 <size> 作为要发送的数据字节数
		Timeout  int `json:"timeout"`  //  time to wait for response
	}
	if err := json.Unmarshal(msgByte, &data); err != nil {
		logger.Warnf("ws processor parseClient err: %s", err.Error())
		return "", fmt.Errorf("query data structure error")
	}

	command := []string{"ping"}
	// 命令字符串高优先级
	if data.Command != "" {
		command = append(command, data.Command)
		command = append(command, "\n")
		return strings.Join(command, " "), nil
	}

	// Options
	if data.Interval > 0 {
		command = append(command, fmt.Sprintf("-i %d", data.Interval))
	}
	if data.TTL > 0 {
		command = append(command, fmt.Sprintf("-t %d", data.TTL))
	}
	if data.Cunt > 0 {
		command = append(command, fmt.Sprintf("-c %d", data.Cunt))
	}
	if data.Size > 0 {
		command = append(command, fmt.Sprintf("-s %d", data.Size))
	}
	if data.Timeout > 0 {
		command = append(command, fmt.Sprintf("-w %d", data.Timeout))
	}

	command = append(command, data.DesAddr)
	command = append(command, "\n")
	return strings.Join(command, " "), nil
}
