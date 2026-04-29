package ws

import (
	"time"
)

// ClientOption 客户端配置选项函数类型
// 使用函数式选项模式，支持链式调用
type ClientOption func(*ClientConfig)

// WithClientSendBufferSize 设置发送队列大小
// 参数：size 发送队列大小，建议值：1000-10000
func WithClientSendBufferSize(size int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.SendBufferSize = size
	}
}

// WithClientWorkers 设置 Worker 数量
// 参数：n Worker数量，用于批量发送
func WithClientWorkers(n int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.Workers = n
	}
}

// WithClientBatchSize 设置批量发送大小
// 参数：size 批量大小，达到此数量立即发送
func WithClientBatchSize(size int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.BatchSize = size
	}
}

// WithClientBatchTimeout 设置批量超时
// 参数：timeout 超时时间，超时后立即发送
func WithClientBatchTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.BatchTimeout = timeout
	}
}

// WithClientRateLimit 设置限流速率
// 参数：rate 每秒允许的请求数
func WithClientRateLimit(rate float64) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.RateLimit = rate
	}
}

// WithClientBackoff 设置背压阈值
// 参数：
//
//	high: 高水位，达到此值开始限流
//	low: 低水位，降到此值恢复正常
func WithClientBackoff(high, low int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.BackoffHigh = high
		cfg.BackoffLow = low
	}
}

// WithClientDialTimeout 设置连接超时
// 参数：timeout 连接超时时间
func WithClientDialTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.DialTimeout = timeout
	}
}

// WithClientReadTimeout 设置读取超时
// 参数：timeout 读取超时时间
func WithClientReadTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.ReadTimeout = timeout
	}
}

// WithClientWriteTimeout 设置写入超时
// 参数：timeout 写入超时时间
func WithClientWriteTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.WriteTimeout = timeout
	}
}

// WithClientMaxPendingRequests 设置最大 pending 请求数
// 参数：n 最大 pending 请求数，用于流量控制
func WithClientMaxPendingRequests(n int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.MaxPendingRequests = n
	}
}
