package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"github.com/tsmask/go-oam/modules/tool/model"
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

// Session 接收ping终端交互业务处理
func (s Ping) Session(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	switch req.Type {
	case "close":
		conn.Close()
		return
	case "probing":
		// Ping会话消息接收写入会话
		if command, err := s.parseOptions(req.Data); command != "" && err == nil {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			if _, err := localClientSession.Write(command); err != nil {
				conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, err.Error(), nil)
			}
		}
	case "ctrl-c":
		// 模拟按下 Ctrl+C
		localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
		if _, err := localClientSession.Write("\u0003\n"); err != nil {
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, err.Error(), nil)
		}
	case "resize":
		// 会话窗口重置
		var data struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		msgByte, _ := json.Marshal(req.Data)
		if err := json.Unmarshal(msgByte, &data); err == nil {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			localClientSession.WindowChange(data.Cols, data.Rows)
		}
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type %s not supported", req.Type), nil)
		return
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
		return "", fmt.Errorf("query data structure error, %s", err.Error())

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
