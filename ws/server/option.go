package server

import "time"

// ServerOption 服务端配置选项
type ServerOption func(*ServerConfig)

// WithServerMaxConns 设置最大连接数
func WithServerMaxConns(n int) ServerOption {
	return func(cfg *ServerConfig) { cfg.MaxConns = n }
}

// WithServerSendBufferSize 设置每连接发送缓冲区大小
func WithServerSendBufferSize(n int) ServerOption {
	return func(cfg *ServerConfig) { cfg.SendBufferSize = n }
}

// WithServerWorkerPoolSize 设置 Handler 并发工作池大小，0 同步执行
func WithServerWorkerPoolSize(n int) ServerOption {
	return func(cfg *ServerConfig) { cfg.WorkerPoolSize = n }
}

// WithServerJobQueueSize 设置任务队列大小
func WithServerJobQueueSize(n int) ServerOption {
	return func(cfg *ServerConfig) { cfg.JobQueueSize = n }
}

// WithServerHeartbeat 设置心跳间隔，0 禁用
func WithServerHeartbeat(d time.Duration) ServerOption {
	return func(cfg *ServerConfig) { cfg.Heartbeat = d }
}

// WithServerRateLimit 设置限流速率（每秒请求数）
func WithServerRateLimit(rate float64) ServerOption {
	return func(cfg *ServerConfig) { cfg.RateLimit = rate }
}

// WithServerAllowedOrigins 设置允许的跨域来源验证函数
func WithServerAllowedOrigins(fn func(origin string) bool) ServerOption {
	return func(cfg *ServerConfig) { cfg.AllowedOriginFunc = fn }
}

// WithServerMaxMessageSize 设置最大消息大小（字节），0 不限制
func WithServerMaxMessageSize(size int) ServerOption {
	return func(cfg *ServerConfig) { cfg.MaxMessageSize = size }
}
