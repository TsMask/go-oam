package ws

import (
	"time"
	"github.com/tsmask/go-oam/ws/client"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/server"
)

// ============================================================================
// 类型别名 — 保持外部调用不变
// ============================================================================

type (
	// 服务端类型
	Server       = server.Server
	ServerConfig = server.ServerConfig
	ConnManager  = server.ConnManager
	Conn         = server.Conn
	Handler      = server.Handler
	Middleware   = server.Middleware
	ServerOption = server.ServerOption

	// 客户端类型
	Client       = client.Client
	ClientConfig = client.ClientConfig
	State        = client.State
	ClientOption = client.ClientOption
)

// ============================================================================
// 状态常量
// ============================================================================

const (
	StateInit          = client.StateInit
	StateConnecting    = client.StateConnecting
	StateConnected     = client.StateConnected
	StateDisconnecting = client.StateDisconnecting
	StateDisconnected  = client.StateDisconnected
	StateReconnecting  = client.StateReconnecting
	StateFailed        = client.StateFailed
)

// ============================================================================
// 错误定义
// ============================================================================

var (
	// 服务端错误
	ErrSendFull = server.ErrSendFull

	// 客户端错误
	ErrTimeout        = client.ErrTimeout
	ErrClientClosed   = client.ErrClientClosed
	ErrConnectionLost = client.ErrConnectionLost
	ErrRateLimited    = client.ErrRateLimited
	ErrInvalidState   = client.ErrInvalidState
)

// ============================================================================
// 构造函数
// ============================================================================

// NewServer 创建 WebSocket 服务端
func NewServer(codec codec.Codec, opts ...ServerOption) *Server {
	return server.NewServer(codec, opts...)
}

// NewClient 创建 WebSocket 客户端
func NewClient(url string, codec codec.Codec, opts ...ClientOption) *Client {
	return client.NewClient(url, codec, opts...)
}

// ============================================================================
// 服务端选项函数
// ============================================================================

// WithServerMaxConns 设置最大连接数
func WithServerMaxConns(n int) ServerOption { return server.WithServerMaxConns(n) }

// WithServerSendBufferSize 设置每连接发送缓冲区大小
func WithServerSendBufferSize(n int) ServerOption { return server.WithServerSendBufferSize(n) }

// WithServerWorkerPoolSize 设置 Handler 并发工作池大小，0 同步执行
func WithServerWorkerPoolSize(n int) ServerOption { return server.WithServerWorkerPoolSize(n) }

// WithServerJobQueueSize 设置任务队列大小
func WithServerJobQueueSize(n int) ServerOption { return server.WithServerJobQueueSize(n) }

// WithServerHeartbeat 设置心跳间隔，0 禁用
func WithServerHeartbeat(d time.Duration) ServerOption { return server.WithServerHeartbeat(d) }

// WithServerRateLimit 设置限流速率（每秒请求数）
func WithServerRateLimit(rate float64) ServerOption { return server.WithServerRateLimit(rate) }

// WithServerAllowedOrigins 设置允许的跨域来源验证函数
func WithServerAllowedOrigins(fn func(origin string) bool) ServerOption {
	return server.WithServerAllowedOrigins(fn)
}

// WithServerMaxMessageSize 设置最大消息大小（字节），0 不限制
func WithServerMaxMessageSize(size int) ServerOption { return server.WithServerMaxMessageSize(size) }

// ============================================================================
// 客户端选项函数
// ============================================================================

// WithClientSendBufferSize 设置发送队列大小
func WithClientSendBufferSize(size int) ClientOption { return client.WithClientSendBufferSize(size) }

// WithClientDialTimeout 设置连接建立超时，默认 30s
func WithClientDialTimeout(timeout time.Duration) ClientOption {
	return client.WithClientDialTimeout(timeout)
}

// WithClientRequestTimeout 设置请求响应超时，默认 60s
func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return client.WithClientRequestTimeout(timeout)
}

// WithClientMaxPendingRequests 设置最大 pending 请求数
func WithClientMaxPendingRequests(n int) ClientOption { return client.WithClientMaxPendingRequests(n) }

// WithClientAutoReconnect 设置是否启用自动重连
func WithClientAutoReconnect(enabled bool) ClientOption { return client.WithClientAutoReconnect(enabled) }

// WithClientMaxReconnectAttempts 设置最大重连次数，默认 10
func WithClientMaxReconnectAttempts(n int) ClientOption {
	return client.WithClientMaxReconnectAttempts(n)
}
