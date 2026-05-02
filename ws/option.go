package ws

import (
	"time"
)

// ============================================================================
// Option 类型定义
// ============================================================================

// ServerOption 服务端配置选项
type ServerOption func(*serverConfig)

// ClientOption 客户端配置选项
type ClientOption func(*ClientConfig)

// ============================================================================
// Server Options
// ============================================================================

// WithServerMaxConns 设置最大连接数
func WithServerMaxConns(n int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.maxConns = n
	}
}

// WithServerSendBufferSize 设置发送缓冲区大小
func WithServerSendBufferSize(n int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.sendBufferSize = n
	}
}

// WithServerRecvBufferSize 设置接收缓冲区大小
func WithServerRecvBufferSize(n int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.recvBufferSize = n
	}
}

// WithServerWorkerPoolSize 设置 Worker 池大小
func WithServerWorkerPoolSize(n int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.workerPoolSize = n
	}
}

// WithServerJobQueueSize 设置任务队列大小
func WithServerJobQueueSize(n int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.jobQueueSize = n
	}
}

// WithServerBatchEnabled 设置是否启用批量发送
func WithServerBatchEnabled(enabled bool) ServerOption {
	return func(cfg *serverConfig) {
		cfg.batchEnabled = enabled
	}
}

// WithServerBatchSize 设置批量发送大小
func WithServerBatchSize(n int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.batchSize = n
	}
}

// WithServerBatchTimeout 设置批量超时
func WithServerBatchTimeout(d time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		cfg.batchTimeout = d
	}
}

// WithServerHeartbeat 设置心跳间隔和超时
func WithServerHeartbeat(interval, timeout time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		cfg.heartbeatInterval = interval
		cfg.heartbeatTimeout = timeout
	}
}

// WithServerRateLimit 设置限流速率
func WithServerRateLimit(rate float64) ServerOption {
	return func(cfg *serverConfig) {
		cfg.rateLimit = rate
	}
}

// WithServerMetrics 设置是否启用指标收集
func WithServerMetrics(enabled bool) ServerOption {
	return func(cfg *serverConfig) {
		cfg.enableMetrics = enabled
	}
}

// WithServerAllowedOrigins 设置允许的跨域来源验证函数
// 如果不配置，默认允许所有跨域请求
// origin 参数为请求的 Origin 头值
func WithServerAllowedOrigins(fn func(origin string) bool) ServerOption {
	return func(cfg *serverConfig) {
		cfg.allowedOriginFunc = fn
	}
}

// WithServerMaxMessageSize 设置最大消息大小（字节）
// 默认为 0，表示不限制
func WithServerMaxMessageSize(size int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.maxMessageSize = size
	}
}

// ============================================================================
// Client Options
// ============================================================================

// WithClientSendBufferSize 设置发送队列大小
func WithClientSendBufferSize(size int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.SendBufferSize = size
	}
}

// WithClientWorkers 设置 Worker 数量
func WithClientWorkers(n int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.Workers = n
	}
}

// WithClientBatchSize 设置批量发送大小
func WithClientBatchSize(size int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.BatchSize = size
	}
}

// WithClientBatchTimeout 设置批量超时
func WithClientBatchTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.BatchTimeout = timeout
	}
}

// WithClientRateLimit 设置限流速率
func WithClientRateLimit(rate float64) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.RateLimit = rate
	}
}

// WithClientBackoff 设置背压阈值
func WithClientBackoff(high, low int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.BackoffHigh = high
		cfg.BackoffLow = low
	}
}

// WithClientDialTimeout 设置连接超时
func WithClientDialTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.DialTimeout = timeout
	}
}

// WithClientReadTimeout 设置读取超时
func WithClientReadTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.ReadTimeout = timeout
	}
}

// WithClientWriteTimeout 设置写入超时
func WithClientWriteTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.WriteTimeout = timeout
	}
}

// WithClientMaxPendingRequests 设置最大 pending 请求数
func WithClientMaxPendingRequests(n int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.MaxPendingRequests = n
	}
}

// WithClientAutoReconnect 设置是否启用自动重连
func WithClientAutoReconnect(enabled bool) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.AutoReconnect = enabled
	}
}

// WithClientReconnectDelay 设置重连延迟
func WithClientReconnectDelay(delay time.Duration) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.ReconnectDelay = delay
	}
}
