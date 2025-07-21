package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/logger"
	"github.com/tsmask/go-oam/framework/ws"
	wsModel "github.com/tsmask/go-oam/modules/ws/model"
	wsService "github.com/tsmask/go-oam/modules/ws/service"
)

// 实例化服务层 IPerf 结构体
var NewIPerf = &IPerf{}

// IPerf 网络性能测试工具 服务层处理
type IPerf struct{}

// Version 查询版本信息
func (s IPerf) Version(version string) (string, error) {
	if version != "V2" && version != "V3" {
		return "", fmt.Errorf("iperf version is required V2 or V3")
	}
	cmdStr := "iperf3 --version"
	if version == "V2" {
		cmdStr = "iperf -v"
	}

	// 检查是否安装iperf
	output, err := cmd.Exec(cmdStr)
	if err != nil {
		if version == "V2" { // V2 版本
			return strings.TrimSpace(strings.TrimPrefix(output, "stderr: ")), nil
		}
		return "", fmt.Errorf("iperf %s not install", version)
	}
	return strings.TrimSpace(output), err
}

// Run 接收IPerf3终端交互业务处理
func (s IPerf) Run(conn *ws.ServerConn, msg []byte) {
	var reqMsg wsModel.WSRequest
	if err := json.Unmarshal(msg, &reqMsg); err != nil {
		wsService.SendErr(conn, "", "message format json error")
		return
	}

	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		wsService.SendErr(conn, "", "message requestId is required")
		return
	}

	switch reqMsg.Type {
	case "close":
		conn.Close()
		return
	case "ping", "PING":
		conn.Pong()
		wsService.SendOK(conn, reqMsg.RequestID, "PONG")
		return
	case "iperf":
		// SSH会话消息接收写入会话
		if command, err := s.parseOptions(reqMsg.Data); command != "" && err == nil {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			if _, err := localClientSession.Write(command); err != nil {
				wsService.SendErr(conn, reqMsg.RequestID, err.Error())
			}
		}
	case "ctrl-c":
		// 模拟按下 Ctrl+C
		localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
		if _, err := localClientSession.Write("\u0003\n"); err != nil {
			wsService.SendErr(conn, reqMsg.RequestID, err.Error())
		}
	case "resize":
		// 会话窗口重置
		var data struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		}
		msgByte, _ := json.Marshal(reqMsg.Data)
		if err := json.Unmarshal(msgByte, &data); err == nil {
			localClientSession := conn.GetAnyConn().(*cmd.LocalClientSession)
			localClientSession.WindowChange(data.Cols, data.Rows)
		}
	default:
		wsService.SendErr(conn, reqMsg.RequestID, fmt.Sprintf("message type %s not supported", reqMsg.Type))
		return
	}
}

// parseOptions 解析拼装iperf3命令 iperf [-s|-c host] [options]
func (s IPerf) parseOptions(reqData any) (string, error) {
	msgByte, _ := json.Marshal(reqData)
	var data struct {
		Command string `json:"command"` // 命令字符串
		Version string `json:"version"` // 服务版本，默认V3
		Mode    string `json:"mode"`    // 服务端或客户端，默认客户端client
		Host    string `json:"host"`    // 客户端连接到的服务端IP地址
		// Server or Client
		Port     int `json:"port"`     // 服务端口
		Interval int `json:"interval"` // 每次报告之间的时间间隔，单位为秒
		// Server
		OneOff bool `json:"oneOff"` // 只进行一次连接
		// Client
		UDP      bool   `json:"udp"`      // use UDP rather than TCP
		Time     int    `json:"time"`     // 以秒为单位的传输时间（默认为 10 秒）
		Reverse  bool   `json:"reverse"`  // 以反向模式运行（服务器发送，客户端接收）
		Window   string `json:"window"`   // 设置窗口大小/套接字缓冲区大小
		Parallel int    `json:"parallel"` // 运行的并行客户端流数量
		Bitrate  int    `json:"bitrate"`  //  以比特/秒为单位（0 表示无限制）
	}
	if err := json.Unmarshal(msgByte, &data); err != nil {
		logger.Warnf("ws processor parseClient err: %s", err.Error())
		return "", fmt.Errorf("query data structure error")
	}
	if data.Version != "V3" && data.Version != "V2" {
		return "", fmt.Errorf("query data version support V3 or V2")
	}

	command := []string{"iperf3"}
	if data.Version == "V2" {
		command = []string{"iperf"}
	}
	// 命令字符串高优先级
	if data.Command != "" {
		command = append(command, data.Command)
		command = append(command, "\n")
		return strings.Join(command, " "), nil
	}

	if data.Mode != "client" && data.Mode != "server" {
		return "", fmt.Errorf("query data mode support client or server")
	}
	if data.Mode == "client" && data.Host == "" {
		return "", fmt.Errorf("query data client host empty")
	}

	if data.Mode == "client" {
		command = append(command, "-c")
		command = append(command, data.Host)
		// Client
		if data.UDP {
			command = append(command, "-u")
		}
		if data.Time > 0 {
			command = append(command, fmt.Sprintf("-t %d", data.Time))
		}
		if data.Bitrate > 0 {
			command = append(command, fmt.Sprintf("-b %d", data.Bitrate))
		}
		if data.Parallel > 0 {
			command = append(command, fmt.Sprintf("-P %d", data.Parallel))
		}
		if data.Reverse {
			command = append(command, "-R")
		}
		if data.Window != "" {
			command = append(command, fmt.Sprintf("-w %s", data.Window))
		}
	}
	if data.Mode == "server" {
		command = append(command, "-s")
		// Server
		if data.OneOff {
			command = append(command, "-1")
		}
	}

	// Server or Client
	if data.Port > 0 {
		command = append(command, fmt.Sprintf("-p %d", data.Port))
	}
	if data.Interval > 0 {
		command = append(command, fmt.Sprintf("-i %d", data.Interval))
	}
	command = append(command, "\n")
	return strings.Join(command, " "), nil
}
