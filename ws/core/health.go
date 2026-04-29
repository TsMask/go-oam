package core

import (
	"sync/atomic"
	"time"
)

// HealthChecker 健康检查器
// 用于检测死连接，超时自动断开
type HealthChecker struct {
	interval  time.Duration       // 检查间隔
	timeout   time.Duration       // 超时时间
	onFailure func(connID string) // 超时回调
}

// NewHealthChecker 创建健康检查器
// 参数：
//
//	interval: 检查间隔
//	timeout: 超时时间
//	onFailure: 超时时回调
func NewHealthChecker(interval, timeout time.Duration, onFailure func(connID string)) *HealthChecker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if onFailure == nil {
		onFailure = func(connID string) {}
	}
	return &HealthChecker{
		interval:  interval,
		timeout:   timeout,
		onFailure: onFailure,
	}
}

// Check 检查是否超时
// 参数：lastActive 上次活跃时间戳（毫秒）
// 返回：未超时返回true，超时返回false
func (hc *HealthChecker) Check(lastActive int64) bool {
	if lastActive == 0 {
		return true
	}
	return time.Since(time.UnixMilli(lastActive)) <= hc.timeout
}

// Start 启动健康检查
// 参数：
//
//	connID: 连接ID
//	lastActive: 原子活跃时间
//	closeChan: 关闭信号通道
func (hc *HealthChecker) Start(connID string, lastActive *atomic.Int64, closeChan <-chan struct{}) {
	ticker := time.NewTicker(hc.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !hc.Check(lastActive.Load()) {
					hc.onFailure(connID)
					ticker.Stop()
					return
				}
			case <-closeChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// HealthMetrics 健康检查指标收集器
type HealthMetrics struct {
	checkCount      atomic.Int64 // 检查次数
	failureCount    atomic.Int64 // 失败次数
	lastCheckTime   atomic.Int64 // 上次检查时间
	lastFailureTime atomic.Int64 // 上次失败时间
	avgResponseTime atomic.Int64 // 平均响应时间
	responseCount   atomic.Int64 // 响应次数
}

// NewHealthMetrics 创建健康指标收集器
func NewHealthMetrics() *HealthMetrics {
	return &HealthMetrics{}
}

// RecordCheck 记录检查
func (hm *HealthMetrics) RecordCheck() {
	hm.checkCount.Add(1)
	hm.lastCheckTime.Store(time.Now().UnixMilli())
}

// RecordFailure 记录失败
func (hm *HealthMetrics) RecordFailure() {
	hm.failureCount.Add(1)
	hm.lastFailureTime.Store(time.Now().UnixMilli())
}

// RecordResponseTime 记录响应时间
func (hm *HealthMetrics) RecordResponseTime(duration time.Duration) {
	oldAvg := hm.avgResponseTime.Load()
	count := hm.responseCount.Add(1)
	newAvg := (oldAvg*(count-1) + duration.Milliseconds()) / count
	hm.avgResponseTime.Store(newAvg)
}

// CheckCount 获取检查次数
func (hm *HealthMetrics) CheckCount() int64 {
	return hm.checkCount.Load()
}

// FailureCount 获取失败次数
func (hm *HealthMetrics) FailureCount() int64 {
	return hm.failureCount.Load()
}

// FailureRate 获取失败率
func (hm *HealthMetrics) FailureRate() float64 {
	count := hm.checkCount.Load()
	if count == 0 {
		return 0
	}
	return float64(hm.failureCount.Load()) / float64(count)
}

// LastCheckTime 获取上次检查时间
func (hm *HealthMetrics) LastCheckTime() time.Time {
	return time.UnixMilli(hm.lastCheckTime.Load())
}

// LastFailureTime 获取上次失败时间
func (hm *HealthMetrics) LastFailureTime() time.Time {
	return time.UnixMilli(hm.lastFailureTime.Load())
}

// AvgResponseTime 获取平均响应时间
func (hm *HealthMetrics) AvgResponseTime() time.Duration {
	return time.Duration(hm.avgResponseTime.Load()) * time.Millisecond
}

// ConnectionHealth 连接健康状态跟踪器
// 每个连接一个实例
type ConnectionHealth struct {
	ID             string       // 连接ID
	LastActive     atomic.Int64 // 上次活跃时间
	LastPingTime   atomic.Int64 // 上次Ping时间
	LastPongTime   atomic.Int64 // 上次Pong时间
	FailureCount   atomic.Int64 // 失败次数
	TotalRequests  atomic.Int64 // 请求总数
	TotalResponses atomic.Int64 // 响应总数
	BytesSent      atomic.Int64 // 发送字节数
	BytesReceived  atomic.Int64 // 接收字节数
}

// NewConnectionHealth 创建连接健康状态跟踪器
func NewConnectionHealth(id string) *ConnectionHealth {
	return &ConnectionHealth{
		ID: id,
	}
}

// UpdateLastActive 更新活跃时间
func (ch *ConnectionHealth) UpdateLastActive() {
	ch.LastActive.Store(time.Now().UnixMilli())
}

// RecordPing 记录Ping
func (ch *ConnectionHealth) RecordPing() {
	ch.LastPingTime.Store(time.Now().UnixMilli())
}

// RecordPong 记录Pong
func (ch *ConnectionHealth) RecordPong() {
	ch.LastPongTime.Store(time.Now().UnixMilli())
}

// RecordRequest 记录请求
func (ch *ConnectionHealth) RecordRequest() {
	ch.TotalRequests.Add(1)
	ch.UpdateLastActive()
}

// RecordResponse 记录响应
func (ch *ConnectionHealth) RecordResponse() {
	ch.TotalResponses.Add(1)
	ch.UpdateLastActive()
}

// RecordBytesSent 记录发送字节数
func (ch *ConnectionHealth) RecordBytesSent(n int) {
	ch.BytesSent.Add(int64(n))
}

// RecordBytesReceived 记录接收字节数
func (ch *ConnectionHealth) RecordBytesReceived(n int) {
	ch.BytesReceived.Add(int64(n))
}

// IsHealthy 检查连接是否健康
func (ch *ConnectionHealth) IsHealthy(timeout time.Duration) bool {
	if ch.LastActive.Load() == 0 {
		return true
	}
	return time.Since(time.UnixMilli(ch.LastActive.Load())) <= timeout
}

// TotalFailures 获取失败次数
func (ch *ConnectionHealth) TotalFailures() int64 {
	return ch.FailureCount.Load()
}
