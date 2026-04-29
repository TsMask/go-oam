package ws

import (
	"time"
)

// ServerOption 服务端配置选项函数类型
// 使用函数式选项模式，支持链式调用
type ServerOption func(*ServerConfig)

// WithServerMaxConns 设置最大连接数
func WithServerMaxConns(n int) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.MaxConns = n
	}
}

// WithServerSendBufferSize 设置发送缓冲区大小
func WithServerSendBufferSize(n int) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.SendBufferSize = n
	}
}

// WithServerRecvBufferSize 设置接收缓冲区大小
func WithServerRecvBufferSize(n int) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.RecvBufferSize = n
	}
}

// WithServerWorkerPoolSize 设置 Worker 池大小
func WithServerWorkerPoolSize(n int) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.WorkerPoolSize = n
	}
}

// WithServerJobQueueSize 设置任务队列大小
func WithServerJobQueueSize(n int) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.JobQueueSize = n
	}
}

// WithServerBatchEnabled 设置是否启用批量发送
func WithServerBatchEnabled(enabled bool) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.BatchEnabled = enabled
	}
}

// WithServerBatchSize 设置批量发送大小
func WithServerBatchSize(n int) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.BatchSize = n
	}
}

// WithServerBatchTimeout 设置批量超时
func WithServerBatchTimeout(d time.Duration) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.BatchTimeout = d
	}
}

// WithServerHeartbeatInterval 设置心跳间隔
func WithServerHeartbeatInterval(d time.Duration) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.HeartbeatInterval = d
	}
}

// WithServerHeartbeatTimeout 设置心跳超时
func WithServerHeartbeatTimeout(d time.Duration) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.HeartbeatTimeout = d
	}
}

// WithServerRateLimit 设置限流速率
func WithServerRateLimit(rate float64) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.RateLimit = rate
	}
}

// WithServerMetrics 设置是否启用指标收集
func WithServerMetrics(enabled bool) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.EnableMetrics = enabled
	}
}
