package core

import (
	"sync/atomic"
	"time"
)

// AdaptiveBackoff 自适应背压控制器
// 根据负载自动调整阈值，比固定阈值更智能
// 包含背压决策和指标收集
type AdaptiveBackoff struct {
	high              int64 // 当前高水位阈值
	low               int64 // 低水位阈值
	backoffMultiplier float64
	recoverMultiplier float64
	current           atomic.Int64 // 当前队列大小
	maxHigh           int64        // 硬上限，防止无限增长

	// 背压指标
	backoffCount    atomic.Int64 // 背压触发次数
	recoverCount    atomic.Int64 // 恢复次数
	lastBackoffTime atomic.Int64 // 上次背压时间
	lastRecoverTime atomic.Int64 // 上次恢复时间
}

// 默认背压参数常量
const (
	DefaultBackoffHigh       = 8000
	DefaultBackoffLow        = 2000
	DefaultMaxBackoffHigh    = 100000 // 10万上限
	DefaultBackoffMultiplier = 1.5
	DefaultRecoverMultiplier = 0.8
)

// NewAdaptiveBackoff 创建自适应背压控制器
func NewAdaptiveBackoff() *AdaptiveBackoff {
	return &AdaptiveBackoff{
		high:              DefaultBackoffHigh,
		low:               DefaultBackoffLow,
		backoffMultiplier: DefaultBackoffMultiplier,
		recoverMultiplier: DefaultRecoverMultiplier,
		maxHigh:           DefaultMaxBackoffHigh,
	}
}

// NewAdaptiveBackoffWithOptions 创建带自定义选项的背压控制器
func NewAdaptiveBackoffWithOptions(high, low int64, maxHigh int64, backoffMul, recoverMul float64) *AdaptiveBackoff {
	if backoffMul <= 0 {
		backoffMul = DefaultBackoffMultiplier
	}
	if recoverMul <= 0 {
		recoverMul = DefaultRecoverMultiplier
	}
	if maxHigh <= 0 {
		maxHigh = DefaultMaxBackoffHigh
	}
	return &AdaptiveBackoff{
		high:              high,
		low:               low,
		backoffMultiplier: backoffMul,
		recoverMultiplier: recoverMul,
		maxHigh:           maxHigh,
	}
}

// Record 记录当前队列大小
func (ab *AdaptiveBackoff) Record(queueSize int) {
	ab.current.Store(int64(queueSize))
}

// RecordInt64 记录当前队列大小（int64版本）
func (ab *AdaptiveBackoff) RecordInt64(queueSize int64) {
	ab.current.Store(queueSize)
}

// MaxHigh 获取当前高水位阈值
func (ab *AdaptiveBackoff) MaxHigh() int64 {
	return atomic.LoadInt64(&ab.high)
}

// ShouldBackoff 检查是否应该触发背压
// 返回：应该背压返回true
func (ab *AdaptiveBackoff) ShouldBackoff() bool {
	val := ab.current.Load()
	for {
		high := atomic.LoadInt64(&ab.high)
		if val > high {
			// 触发背压时，提高阈值，但不超过 maxHigh
			newHigh := int64(float64(high) * ab.backoffMultiplier)
			if newHigh > ab.maxHigh {
				newHigh = ab.maxHigh
			}
			// 使用 CAS 保证原子更新
			if atomic.CompareAndSwapInt64(&ab.high, high, newHigh) {
				ab.backoffCount.Add(1)
				ab.lastBackoffTime.Store(time.Now().UnixMilli())
				return true
			}
			continue // 重试
		}

		// 恢复正常时，降低阈值
		if val < ab.low && high > DefaultBackoffHigh {
			newHigh := int64(float64(high) * ab.recoverMultiplier)
			if newHigh < DefaultBackoffHigh {
				newHigh = DefaultBackoffHigh
			}
			if atomic.CompareAndSwapInt64(&ab.high, high, newHigh) {
				ab.recoverCount.Add(1)
				ab.lastRecoverTime.Store(time.Now().UnixMilli())
				return false
			}
			continue
		}
		return false
	}
}

// Current 获取当前队列大小
func (ab *AdaptiveBackoff) Current() int {
	return int(ab.current.Load())
}

// BackoffCount 获取背压触发次数
func (ab *AdaptiveBackoff) BackoffCount() int64 {
	return ab.backoffCount.Load()
}

// RecoverCount 获取恢复次数
func (ab *AdaptiveBackoff) RecoverCount() int64 {
	return ab.recoverCount.Load()
}

// LastBackoffTime 获取上次背压时间
func (ab *AdaptiveBackoff) LastBackoffTime() time.Time {
	return time.UnixMilli(ab.lastBackoffTime.Load())
}

// LastRecoverTime 获取上次恢复时间
func (ab *AdaptiveBackoff) LastRecoverTime() time.Time {
	return time.UnixMilli(ab.lastRecoverTime.Load())
}
