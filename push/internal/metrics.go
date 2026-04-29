package internal

import (
	"sync"
)

type metric struct {
	accum   float64 // 累积值（收集线程累加到这里）
	sent    float64 // 已发送值（标记前的值）
	initVal float64 // 初始值
	step    float64 // 每次变化量
	minVal  float64 // 最小值
	maxVal  float64 // 最大值
	mu      sync.Mutex
}

type Metrics struct {
	data sync.Map
}

// NewMetrics 创建指标管理器
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Register 注册指标
// max最大值可以用`math.Inf(1)`
func (m *Metrics) Register(name string, init, step, min, max float64) {
	m.data.Store(name, &metric{
		accum:   init,
		sent:    init,
		initVal: init,
		step:    step,
		minVal:  min,
		maxVal:  max,
	})
}

// Set 设置指标值（设置累积值）
func (m *Metrics) Set(name string, value float64) {
	if v, ok := m.data.Load(name); ok {
		mt := v.(*metric)
		mt.mu.Lock()
		mt.accum = value
		mt.mu.Unlock()
	}
}

// Inc 累加指标（增量累加到累积值）
func (m *Metrics) Inc(name string) {
	if v, ok := m.data.Load(name); ok {
		mt := v.(*metric)
		mt.mu.Lock()
		v := mt.accum + mt.step
		if v > mt.maxVal {
			v = mt.maxVal
		}
		mt.accum = v
		mt.mu.Unlock()
	}
}

// Dec 递减指标
func (m *Metrics) Dec(name string) {
	if v, ok := m.data.Load(name); ok {
		mt := v.(*metric)
		mt.mu.Lock()
		v := mt.accum - mt.step
		if v < mt.minVal {
			v = mt.minVal
		}
		mt.accum = v
		mt.mu.Unlock()
	}
}

// IncBy 增量累加
func (m *Metrics) IncBy(name string, delta float64) {
	if v, ok := m.data.Load(name); ok {
		mt := v.(*metric)
		mt.mu.Lock()
		v := mt.accum + delta
		if v > mt.maxVal {
			v = mt.maxVal
		}
		mt.accum = v
		mt.mu.Unlock()
	}
}

// Get 获取当前累积值
func (m *Metrics) Get(name string) float64 {
	if v, ok := m.data.Load(name); ok {
		mt := v.(*metric)
		mt.mu.Lock()
		val := mt.accum
		mt.mu.Unlock()
		return val
	}
	return 0
}

// GetDelta 获取增量（累积值 - 已发送值）
func (m *Metrics) GetDelta(name string) float64 {
	if v, ok := m.data.Load(name); ok {
		mt := v.(*metric)
		mt.mu.Lock()
		val := mt.accum - mt.sent
		mt.mu.Unlock()
		return val
	}
	return 0
}

// Flush 标记触发：获取增量并重置已发送值
// 返回所有指标的增量数据
func (m *Metrics) Flush() map[string]float64 {
	result := make(map[string]float64)
	m.data.Range(func(key, value any) bool {
		mt := value.(*metric)
		mt.mu.Lock()
		result[key.(string)] = mt.accum - mt.sent
		mt.sent = mt.accum
		mt.mu.Unlock()
		return true
	})
	return result
}

// FlushAndReset 标记触发：获取累积值并重置
// 返回所有指标的累积数据，并将 accum/sent 重置到初始值
func (m *Metrics) FlushAndReset() map[string]float64 {
	result := make(map[string]float64)
	m.data.Range(func(key, value any) bool {
		mt := value.(*metric)
		mt.mu.Lock()
		result[key.(string)] = mt.accum
		mt.accum = mt.initVal
		mt.sent = mt.initVal
		mt.mu.Unlock()
		return true
	})
	return result
}

// Snapshot 获取当前累积值快照（不修改）
func (m *Metrics) Snapshot() map[string]float64 {
	result := make(map[string]float64)
	m.data.Range(func(key, value any) bool {
		mt := value.(*metric)
		mt.mu.Lock()
		result[key.(string)] = mt.accum
		mt.mu.Unlock()
		return true
	})
	return result
}

// Clear 重置所有指标到初始值
func (m *Metrics) Clear() {
	m.data.Range(func(key, value any) bool {
		mt := value.(*metric)
		mt.mu.Lock()
		mt.accum = mt.initVal
		mt.sent = mt.initVal
		mt.mu.Unlock()
		return true
	})
}

// Count 获取指标数量
func (m *Metrics) Count() int {
	count := 0
	m.data.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

// Keys 获取所有指标名
func (m *Metrics) Keys() []string {
	keys := make([]string, 0, 16)
	m.data.Range(func(key, value any) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}
