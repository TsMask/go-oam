package iperf

import (
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/pkg/cmd"
)

// Terminal iperf 命令行终端参数，用于构建和执行 iperf/iperf3 网络性能测试命令。
// 支持 V2（iperf）和 V3（iperf3）两个版本，支持 client/server 模式。
// 当 Command 不为空时直接执行该命令，忽略其余参数。
type Terminal struct {
	Command  string `json:"command"`  // 自定义原始命令（非空时跳过参数拼装，直接执行）
	Mode     string `json:"mode"`     // 运行模式："client" 或 "server"
	Host     string `json:"host"`     // 目标主机地址（client 模式必填）
	Port     int    `json:"port"`     // 端口号，默认由 iperf 决定
	Interval int    `json:"interval"` // 统计输出间隔，单位秒
	OneOff   bool   `json:"oneOff"`   // server 模式下接受一次连接后自动退出（-1）
	UDP      bool   `json:"udp"`      // 使用 UDP 协议（默认 TCP）
	Time     int    `json:"time"`     // 传输持续时间，单位秒（client 模式）
	Reverse  bool   `json:"reverse"`  // 反向模式：server 端发送，client 端接收（-R）
	Window   string `json:"window"`   // TCP 窗口大小（如 "256K"）
	Parallel int    `json:"parallel"` // 并行连接数（-P）
	Bitrate  int    `json:"bitrate"`  // 目标带宽，单位 bit/s（UDP 模式常用）
}

// Version 查询本机 iperf 版本信息。
// version 参数仅支持 "V2" 或 "V3"，分别对应 iperf 和 iperf3。
func (t *Terminal) Version(version string) (string, error) {
	if version != "V2" && version != "V3" {
		return "", fmt.Errorf("iperf version is required V2 or V3")
	}
	cmdStr := "iperf3 --version"
	if version == "V2" {
		cmdStr = "iperf -v"
	}
	output, err := cmd.Exec(cmdStr)
	if err != nil {
		if version == "V2" {
			return strings.TrimSpace(strings.TrimPrefix(output, "stderr: ")), nil
		}
		return "", fmt.Errorf("iperf %s not install", version)
	}
	return strings.TrimSpace(output), err
}

// ParseOptions 根据终端参数拼装 iperf 命令行字符串。
// version 参数指定协议版本："V2" 使用 iperf，"V3" 使用 iperf3。
// 当 Command 非空时直接返回该命令；否则按 Mode 及各字段拼装参数。
// client 模式下 Host 不能为空；mode 仅支持 "client" 和 "server"。
// 返回值末尾带换行符，适配交互式终端写入。
func (t *Terminal) ParseOptions(version string) (string, error) {
	if version != "V3" && version != "V2" {
		return "", fmt.Errorf("query data version support V3 or V2")
	}
	command := []string{"iperf3"}
	if version == "V2" {
		command = []string{"iperf"}
	}
	if t.Command != "" {
		command = append(command, t.Command)
		command = append(command, "\n")
		return strings.Join(command, " "), nil
	}
	if t.Mode != "client" && t.Mode != "server" {
		return "", fmt.Errorf("query data mode support client or server")
	}
	if t.Mode == "client" && t.Host == "" {
		return "", fmt.Errorf("query data client host empty")
	}
	if t.Mode == "client" {
		command = append(command, "-c", t.Host)
		if t.UDP {
			command = append(command, "-u")
		}
		if t.Time > 0 {
			command = append(command, fmt.Sprintf("-t %d", t.Time))
		}
		if t.Bitrate > 0 {
			command = append(command, fmt.Sprintf("-b %d", t.Bitrate))
		}
		if t.Parallel > 0 {
			command = append(command, fmt.Sprintf("-P %d", t.Parallel))
		}
		if t.Reverse {
			command = append(command, "-R")
		}
		if t.Window != "" {
			command = append(command, fmt.Sprintf("-w %s", t.Window))
		}
	}
	if t.Mode == "server" {
		command = append(command, "-s")
		if t.OneOff {
			command = append(command, "-1")
		}
	}
	if t.Port > 0 {
		command = append(command, fmt.Sprintf("-p %d", t.Port))
	}
	if t.Interval > 0 {
		command = append(command, fmt.Sprintf("-i %d", t.Interval))
	}
	command = append(command, "\n")
	return strings.Join(command, " "), nil
}
