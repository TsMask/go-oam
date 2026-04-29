package core

import (
	"sync/atomic"
	"time"
)

// AdaptiveBackoff 自适应背压控制器
// 根据负载自动调整阈值，比固定阈值更智能
type AdaptiveBackoff struct {
	high              int          // 高水位阈值
	low               int          // 低水位阈值
	backoffMultiplier float64      // 背压时阈值增长倍数
	recoverMultiplier float64      // 恢复时阈值缩小倍数
	current           atomic.Int64 // 当前队列大小
}

// NewAdaptiveBackoff 创建自适应背压控制器
func NewAdaptiveBackoff() *AdaptiveBackoff {
	return &AdaptiveBackoff{
		high:              8000,
		low:               2000,
		backoffMultiplier: 1.5,
		recoverMultiplier: 0.8,
	}
}

// Record 记录当前队列大小
func (ab *AdaptiveBackoff) Record(queueSize int) {
	ab.current.Store(int64(queueSize))
}

// ShouldBackoff 检查是否应该触发背压
// 返回：应该背压返回true
func (ab *AdaptiveBackoff) ShouldBackoff() bool {
	val := int(ab.current.Load())
	if val > ab.high {
		// 触发背压时，提高阈值
		ab.high = int(float64(val) * ab.backoffMultiplier)
		return true
	}
	if val < ab.low && ab.high > 8000 {
		// 恢复正常时，降低阈值
		ab.high = int(float64(ab.high) * ab.recoverMultiplier)
		if ab.high < 8000 {
			ab.high = 8000
		}
	}
	return false
}

// Current 获取当前队列大小
func (ab *AdaptiveBackoff) Current() int {
	return int(ab.current.Load())
}

// BackoffMetrics 背压指标收集器
type BackoffMetrics struct {
	backoffCount    atomic.Int64 // 背压触发次数
	recoverCount    atomic.Int64 // 恢复次数
	lastBackoffTime atomic.Int64 // 上次背压时间
	lastRecoverTime atomic.Int64 // 上次恢复时间
}

// NewBackoffMetrics 创建背压指标收集器
func NewBackoffMetrics() *BackoffMetrics {
	return &BackoffMetrics{}
}

// RecordBackoff 记录背压触发
func (bm *BackoffMetrics) RecordBackoff() {
	bm.backoffCount.Add(1)
	bm.lastBackoffTime.Store(time.Now().UnixMilli())
}

// RecordRecover 记录恢复
func (bm *BackoffMetrics) RecordRecover() {
	bm.recoverCount.Add(1)
	bm.lastRecoverTime.Store(time.Now().UnixMilli())
}

// BackoffCount 获取背压触发次数
func (bm *BackoffMetrics) BackoffCount() int64 {
	return bm.backoffCount.Load()
}

// RecoverCount 获取恢复次数
func (bm *BackoffMetrics) RecoverCount() int64 {
	return bm.recoverCount.Load()
}

// LastBackoffTime 获取上次背压时间
func (bm *BackoffMetrics) LastBackoffTime() time.Time {
	return time.UnixMilli(bm.lastBackoffTime.Load())
}

// LastRecoverTime 获取上次恢复时间
func (bm *BackoffMetrics) LastRecoverTime() time.Time {
	return time.UnixMilli(bm.lastRecoverTime.Load())
}
