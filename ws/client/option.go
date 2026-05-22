package client

import "time"

// ClientOption 客户端配置选项
type ClientOption func(*ClientConfig)

// WithClientSendBufferSize 设置发送队列大小
func WithClientSendBufferSize(size int) ClientOption {
	return func(cfg *ClientConfig) { cfg.SendBufferSize = size }
}

// WithClientDialTimeout 设置连接建立超时，默认 30s
func WithClientDialTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) { cfg.DialTimeout = timeout }
}

// WithClientRequestTimeout 设置请求响应超时，默认 60s
func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) { cfg.RequestTimeout = timeout }
}

// WithClientMaxPendingRequests 设置最大 pending 请求数
func WithClientMaxPendingRequests(n int) ClientOption {
	return func(cfg *ClientConfig) { cfg.MaxPendingRequests = n }
}

// WithClientAutoReconnect 设置是否启用自动重连
func WithClientAutoReconnect(enabled bool) ClientOption {
	return func(cfg *ClientConfig) { cfg.AutoReconnect = enabled }
}

// WithClientMaxReconnectAttempts 设置最大重连次数，默认 10
func WithClientMaxReconnectAttempts(n int) ClientOption {
	return func(cfg *ClientConfig) { cfg.MaxReconnectAttempts = n }
}
