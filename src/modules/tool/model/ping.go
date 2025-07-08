package model

import (
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Ping 探针发包参数
type Ping struct {
	DesAddr  string `json:"desAddr" binding:"required"` // 目的 IP 地址（字符串类型，必填）
	SrcAddr  string `json:"srcAddr"`                    // 源 IP 地址（字符串类型，可选）
	Interval int    `json:"interval"`                   // 发包间隔（整数类型，可选，单位：秒，取值范围：1-60，默认值：1）
	TTL      int    `json:"ttl"`                        // TTL（整数类型，可选，取值范围：1-255，默认值：255）
	Count    int    `json:"count"`                      // 发包数（整数类型，可选，取值范围：1-65535，默认值：5）
	Size     int    `json:"size"`                       // 报文大小（整数类型，可选，取值范围：36-8192，默认值：36）
	Timeout  int    `json:"timeout"`                    // 报文超时时间（整数类型，可选，单位：秒，取值范围：1-60，默认值：2）
}

// setDefaultValue 设置默认值
func (p *Ping) setDefaultValue() {
	if p.Interval < 1 || p.Interval > 10 {
		p.Interval = 1
	}
	if p.TTL < 1 || p.TTL > 255 {
		p.TTL = 255
	}
	if p.Count < 1 || p.Count > 65535 {
		p.Count = 5
	}
	if p.Size < 36 || p.Size > 8192 {
		p.Size = 36
	}
	if p.Timeout < 1 || p.Timeout > 60 {
		p.Timeout = 2
	}
}

// NewPinger ping对象
func (p *Ping) NewPinger() (*probing.Pinger, error) {
	p.setDefaultValue()

	pinger, err := probing.NewPinger(p.DesAddr)
	if err != nil {
		return nil, err
	}
	if p.SrcAddr != "" {
		pinger.Source = p.SrcAddr
	}
	pinger.Interval = time.Duration(p.Interval) * time.Second
	pinger.TTL = p.TTL
	pinger.Count = p.Count
	pinger.Size = p.Size
	pinger.Timeout = time.Duration(p.Timeout) * time.Second

	// 设置特权模式（需要管理员权限）
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}
	return pinger, nil
}
