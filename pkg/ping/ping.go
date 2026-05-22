package ping

import (
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Ping 探针发包参数
type Ping struct {
	DesAddr  string `json:"desAddr" binding:"required"`
	SrcAddr  string `json:"srcAddr"`
	Interval int    `json:"interval"`
	TTL      int    `json:"ttl"`
	Count    int    `json:"count"`
	Size     int    `json:"size"`
	Timeout  int    `json:"timeout"`
}

// setDefaultValue 初始默认值
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

// NewPinger 构造 probing.Pinger 对象
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
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}
	return pinger, nil
}

// Statistics 执行ping并返回统计数据
func (s *Ping) Statistics(ping Ping) (map[string]int64, error) {
	pinger, err := ping.NewPinger()
	if err != nil {
		return nil, err
	}
	if err = pinger.Run(); err != nil {
		return nil, err
	}
	defer pinger.Stop()
	stats := pinger.Statistics()
	data := map[string]int64{
		"minTime":  stats.MinRtt.Microseconds(),
		"maxTime":  stats.MaxRtt.Microseconds(),
		"avgTime":  stats.AvgRtt.Microseconds(),
		"lossRate": int64(stats.PacketLoss),
		"jitter":   stats.StdDevRtt.Microseconds(),
	}
	return data, nil
}
