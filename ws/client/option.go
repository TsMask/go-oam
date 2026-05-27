package client

import "time"

// ClientOption 客户端配置选项
type ClientOption func(*clientConfig)

// clientConfig 客户端内部配置
type clientConfig struct {
	codec                string
	dialTimeout          time.Duration
	autoReconnect        bool
	maxReconnectAttempts int
	heartbeat            time.Duration
}

// WithClientCodec 设置编解码器，支持 "json"/"msgpack"/"protobuf"，默认 "json"
func WithClientCodec(name string) ClientOption {
	return func(cfg *clientConfig) { cfg.codec = name }
}

// WithClientDialTimeout 设置连接建立超时，默认 30s
func WithClientDialTimeout(timeout time.Duration) ClientOption {
	return func(cfg *clientConfig) { cfg.dialTimeout = timeout }
}

// WithClientAutoReconnect 设置是否启用自动重连
func WithClientAutoReconnect(enabled bool) ClientOption {
	return func(cfg *clientConfig) { cfg.autoReconnect = enabled }
}

// WithClientMaxReconnectAttempts 设置最大重连次数，默认 10
func WithClientMaxReconnectAttempts(n int) ClientOption {
	return func(cfg *clientConfig) { cfg.maxReconnectAttempts = n }
}

// WithClientHeartbeat 设置健康检查间隔，默认 15s，0 禁用
func WithClientHeartbeat(interval time.Duration) ClientOption {
	return func(cfg *clientConfig) { cfg.heartbeat = interval }
}
