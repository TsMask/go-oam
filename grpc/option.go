package grpc

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

// serverConfig 内部服务端配置
type serverConfig struct {
	maxConns          int
	sendBufferSize    int
	recvBufferSize    int
	workerPoolSize    int
	jobQueueSize      int
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	rateLimit         float64
}

// ClientConfig 客户端配置
type ClientConfig struct {
	SendBufferSize     int
	Workers            int
	RateLimit          float64
	BackoffHigh        int
	BackoffLow         int
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	MaxPendingRequests int
	AutoReconnect      bool
	ReconnectDelay     time.Duration
	Meta               map[string]string
}

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

// WithServerRateLimit 设置限流速率
func WithServerRateLimit(rate float64) ServerOption {
	return func(cfg *serverConfig) {
		cfg.rateLimit = rate
	}
}

// WithServerHeartbeat 设置心跳间隔和超时
func WithServerHeartbeat(interval, timeout time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		cfg.heartbeatInterval = interval
		cfg.heartbeatTimeout = timeout
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

// WithClientMaxPendingRequests 设置最大 pending 请求数
func WithClientMaxPendingRequests(n int) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.MaxPendingRequests = n
	}
}

// WithMeta 设置客户端元数据
func WithMeta(meta map[string]string) ClientOption {
	return func(cfg *ClientConfig) {
		cfg.Meta = meta
	}
}
