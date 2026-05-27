package server

import "time"

// ServerOption 服务端配置选项
type ServerOption func(*serverConfig)

// serverConfig 服务端内部配置
type serverConfig struct {
	codec             string
	maxConns          int
	sendBufferSize    int
	heartbeat         time.Duration
	allowedOriginFunc func(origin string) bool
	maxMessageSize    int
}

// WithServerCodec 设置编解码器，支持 "json"/"msgpack"/"protobuf"，默认 "json"
func WithServerCodec(name string) ServerOption {
	return func(cfg *serverConfig) { cfg.codec = name }
}

// WithServerMaxConns 设置最大连接数
func WithServerMaxConns(n int) ServerOption {
	return func(cfg *serverConfig) { cfg.maxConns = n }
}

// WithServerSendBufferSize 设置每连接发送缓冲区大小
func WithServerSendBufferSize(n int) ServerOption {
	return func(cfg *serverConfig) { cfg.sendBufferSize = n }
}

// WithServerHeartbeat 设置空闲超时检测间隔，0 禁用
func WithServerHeartbeat(d time.Duration) ServerOption {
	return func(cfg *serverConfig) { cfg.heartbeat = d }
}

// WithServerAllowedOrigins 设置允许的跨域来源验证函数
func WithServerAllowedOrigins(fn func(origin string) bool) ServerOption {
	return func(cfg *serverConfig) { cfg.allowedOriginFunc = fn }
}

// WithServerMaxMessageSize 设置最大消息大小（字节），0 不限制
func WithServerMaxMessageSize(size int) ServerOption {
	return func(cfg *serverConfig) { cfg.maxMessageSize = size }
}
