package ping

import (
	"encoding/json"
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Ping 探针发包参数
type Ping struct {
	DesAddr      string `json:"desAddr"`      // 目的 IP 地址（字符串类型，必填）
	VRFName      string `json:"vrfName"`      // VRF 名称（字符串类型，可选）
	DSCP         int    `json:"dscp"`         // DSCP 优先级（整数类型，可选，取值范围：0-63，默认值：0）
	SrcAddr      string `json:"srcAddr"`      // 源 IP 地址（字符串类型，可选）
	SendInterval int    `json:"sendInterval"` // 发包间隔（整数类型，可选，单位：毫秒，取值范围：10-10000，默认值：1000）
	TTL          int    `json:"ttl"`          // TTL（整数类型，可选，取值范围：1-255，默认值：255）
	SentPkts     int    `json:"sentPkts"`     // 发包数（整数类型，可选，取值范围：1-65535，默认值：5）
	Size         int    `json:"size"`         // 报文大小（整数类型，可选，取值范围：36-8192，默认值：36）
	TimeOut      int    `json:"timeOut"`      // 报文超时时间（整数类型，可选，单位：秒，取值范围：1-60，默认值：2）
}

// CopyFrom 将map复制到当前同key名的结构体
func (p *Ping) CopyFrom(from any) error {
	b, err := json.Marshal(from)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, p); err != nil {
		return err
	}
	return nil
}

// setDefaultValue 设置默认值
func (p *Ping) setDefaultValue() {
	if p.SendInterval < 10 || p.SendInterval > 10000 {
		p.SendInterval = 1000
	}
	if p.TTL < 1 || p.TTL > 255 {
		p.TTL = 255
	}
	if p.SentPkts < 1 || p.SentPkts > 65535 {
		p.SentPkts = 5
	}
	if p.Size < 36 || p.Size > 8192 {
		p.Size = 36
	}
	if p.DSCP < 0 || p.DSCP > 63 {
		p.DSCP = 0
	}
	if p.TimeOut < 1 || p.TimeOut > 60 {
		p.TimeOut = 2
	}
}

// Statistics ping数据结果
func (p *Ping) Statistics() (*probing.Statistics, error) {
	p.setDefaultValue()

	pinger, err := probing.NewPinger(p.DesAddr)
	if err != nil {
		return nil, err
	}
	if p.SrcAddr != "" {
		pinger.Source = p.SrcAddr
	}
	pinger.Interval = time.Duration(p.SendInterval) * time.Millisecond
	pinger.TTL = p.TTL
	pinger.Count = p.SentPkts
	pinger.Size = p.Size
	pinger.Timeout = time.Duration(p.TimeOut) * time.Second

	// 设置特权模式（需要管理员权限）
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}
	err = pinger.Run()
	if err != nil {
		return nil, err
	}

	return pinger.Statistics(), nil
}

// StatsInfo ping基本信息
func (p *Ping) StatsInfo() (map[string]any, error) {
	stats, err := p.Statistics()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"minTime":  stats.MinRtt.Milliseconds(),    // 最小时延（整数类型，可选，单位：毫秒）
		"maxTime":  stats.MaxRtt.Milliseconds(),    // 最大时延（整数类型，可选，单位：毫秒）
		"avgTime":  stats.AvgRtt.Milliseconds(),    // 平均时延（整数类型，可选，单位：毫秒）
		"lossRate": int64(stats.PacketLoss),        // 丢包率（整数类型，可选，单位：%）
		"jitter":   stats.StdDevRtt.Milliseconds(), // 时延抖动（整数类型，可选，单位：毫秒）
	}, nil
}

// StatsInfo ping探测发送的所有往返时间
func (p *Ping) StatsRtt() (map[string][]map[string]any, error) {
	stats, err := p.Statistics()
	if err != nil {
		return nil, err
	}
	data := map[string][]map[string]any{
		// hopList	节点列表
		"hopList": {
			{
				// hopIndex	序号
				"hopIndex": 1,
				// probeList	探测信息列表
				"probeList": []map[string]any{
					{
						"probeIndex": 1,   // probeIndex	探测报文序号
						"hopAddress": "-", // hopAddress	地址
						"probeTime":  0,   // probeTime	探测时长
					},
				},
			},
		},
	}

	rtts := []map[string]any{}
	for i, tts := range stats.Rtts {
		rtts = append(rtts, map[string]any{
			"probeIndex": i + 1,                 // probeIndex	探测报文序号
			"hopAddress": stats.IPAddr.String(), // hopAddress	地址
			"probeTime":  tts.Milliseconds(),    // probeTime	探测时长
		})
	}
	data["hopList"][0]["probeList"] = rtts

	return data, nil
}
