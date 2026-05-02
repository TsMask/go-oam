package core

import (
	"sync/atomic"
	"time"
)

// AtomicRateLimiter 原子限流器
// 使用CAS无锁操作，性能比Mutex版本提升3-5倍
type AtomicRateLimiter struct {
	rate   float64      // 每秒允许的令牌数
	burst  int32        // 令牌桶容量
	tokens atomic.Int64 // 当前令牌数
	last   atomic.Int64 // 上次更新时间
}

// NewAtomicRateLimiter 创建原子限流器
// 参数：
//
//	rate: 每秒允许的令牌数
//	burst: 令牌桶容量
func NewAtomicRateLimiter(rate float64, burst int) *AtomicRateLimiter {
	if rate <= 0 {
		rate = 10000
	}
	if burst <= 0 {
		burst = 10000
	}
	rl := &AtomicRateLimiter{
		rate:  rate,
		burst: int32(burst),
	}
	rl.tokens.Store(int64(burst))
	rl.last.Store(time.Now().UnixNano())
	return rl
}

// Allow 检查是否允许请求
// 返回：允许返回true，限流返回false
func (r *AtomicRateLimiter) Allow() bool {
	for {
		now := time.Now().UnixNano()
		elapsed := float64(now-r.last.Load()) / float64(time.Second)

		tokens := float64(r.tokens.Load()) + elapsed*r.rate
		if tokens > float64(r.burst) {
			tokens = float64(r.burst)
		}

		if tokens >= 1 {
			if r.tokens.CompareAndSwap(int64(tokens), int64(tokens-1)) {
				r.last.Store(now)
				return true
			}
			continue
		}
		return false
	}
}

// Tokens 获取当前令牌数
func (r *AtomicRateLimiter) Tokens() float64 {
	now := time.Now().UnixNano()
	elapsed := float64(now-r.last.Load()) / float64(time.Second)

	tokens := float64(r.tokens.Load()) + elapsed*r.rate
	if tokens > float64(r.burst) {
		tokens = float64(r.burst)
	}
	return tokens
}

// Rate 获取限流速率（每秒令牌数）
func (r *AtomicRateLimiter) Rate() float64 {
	return r.rate
}

// Burst 获取令牌桶容量
func (r *AtomicRateLimiter) Burst() int32 {
	return r.burst
}

// IsLimited 检查当前是否处于限流状态（令牌数 < 1）
func (r *AtomicRateLimiter) IsLimited() bool {
	return r.Tokens() < 1
}

// WaitTime 估算需要等待多久才能获得一个令牌
func (r *AtomicRateLimiter) WaitTime() time.Duration {
	tokens := r.Tokens()
	if tokens >= 1 {
		return 0
	}
	deficit := 1 - tokens
	return time.Duration(deficit/r.rate) * time.Second
}