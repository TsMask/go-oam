package ping

import (
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/pkg/cmd"
)

// Terminal ping 命令行终端参数，用于构建和执行 ping 命令。
// 当 Command 不为空时直接执行该命令，忽略其余参数。
type Terminal struct {
	Command  string `json:"command"`  // 自定义原始命令（非空时跳过参数拼装，直接执行）
	DesAddr  string `json:"desAddr"`  // 目标地址（IP 或域名）
	Interval int    `json:"interval"` // 发包间隔，单位秒，范围 1-10，默认 1
	TTL      int    `json:"ttl"`      // 生存时间，范围 1-255，默认 255
	Cunt     int    `json:"count"`    // 发包次数，范围 1-65535，默认 5
	Size     int    `json:"size"`     // 每包字节数，范围 36-8192，默认 36
	Timeout  int    `json:"timeout"`  // 超时时间，单位秒，范围 1-60，默认 2
}

// Version 查询本机 ping 版本信息。
// 返回 ping -V 的输出内容；若 ping 未安装则返回错误。
func (t Terminal) Version() (string, error) {
	output, err := cmd.Exec("ping -V")
	if err != nil {
		return "", fmt.Errorf("ping not installed")
	}
	return strings.TrimSpace(output), err
}

// ParseOptions 根据终端参数拼装 ping 命令行字符串。
// 当 Command 非空时直接返回该命令；否则按各字段拼装标准 ping 参数。
// 返回值末尾带换行符，适配交互式终端写入。
func (t *Terminal) ParseOptions() (string, error) {
	command := []string{"ping"}
	if t.Command != "" {
		command = append(command, t.Command)
		command = append(command, "\n")
		return strings.Join(command, " "), nil
	}
	if t.Interval > 0 {
		command = append(command, fmt.Sprintf("-i %d", t.Interval))
	}
	if t.TTL > 0 {
		command = append(command, fmt.Sprintf("-t %d", t.TTL))
	}
	if t.Cunt > 0 {
		command = append(command, fmt.Sprintf("-c %d", t.Cunt))
	}
	if t.Size > 0 {
		command = append(command, fmt.Sprintf("-s %d", t.Size))
	}
	if t.Timeout > 0 {
		command = append(command, fmt.Sprintf("-w %d", t.Timeout))
	}
	command = append(command, t.DesAddr)
	command = append(command, "\n")
	return strings.Join(command, " "), nil
}
