package ws

import (
	"time"

	"github.com/tsmask/go-oam/ws/client"
	"github.com/tsmask/go-oam/ws/server"
)

// ============================================================================
// 类型别名 — 保持外部调用不变
// ============================================================================

type (
	// 服务端类型
	Server       = server.Server
	ConnManager  = server.ConnManager
	Conn         = server.Conn
	Handler      = server.Handler
	Middleware   = server.Middleware
	ServerOption = server.ServerOption

	// 客户端类型
	Client       = client.Client
	State        = client.State
	ClientOption = client.ClientOption
)

// ============================================================================
// 状态常量
// ============================================================================

const (
	StateInit         = client.StateInit
	StateConnecting   = client.StateConnecting
	StateConnected    = client.StateConnected
	StateDisconnected = client.StateDisconnected
	StateReconnecting = client.StateReconnecting
	StateFailed       = client.StateFailed
)

// ============================================================================
// 错误定义
// ============================================================================

var (
	// 服务端错误
	ErrSendFull = server.ErrSendFull

	// 客户端错误
	ErrClientClosed   = client.ErrClientClosed
	ErrConnectionLost = client.ErrConnectionLost
	ErrInvalidState   = client.ErrInvalidState
)

// ============================================================================
// 构造函数
// ============================================================================

// NewServer 创建 WebSocket 服务端，默认 JSON 编解码
func NewServer(opts ...ServerOption) *Server {
	return server.NewServer(opts...)
}

// NewClient 创建 WebSocket 客户端，默认 JSON 编解码
func NewClient(url string, opts ...ClientOption) *Client {
	return client.NewClient(url, opts...)
}

// ============================================================================
// 服务端选项函数
// ============================================================================

// WithServerCodec 设置编解码器，支持 "json"/"msgpack"/"protobuf"，默认 "json"
func WithServerCodec(name string) ServerOption { return server.WithServerCodec(name) }

// WithServerMaxConns 设置最大连接数
func WithServerMaxConns(n int) ServerOption { return server.WithServerMaxConns(n) }

// WithServerSendBufferSize 设置每连接发送缓冲区大小
func WithServerSendBufferSize(n int) ServerOption { return server.WithServerSendBufferSize(n) }

// WithServerHeartbeat 设置心跳间隔，0 禁用
func WithServerHeartbeat(d time.Duration) ServerOption { return server.WithServerHeartbeat(d) }

// WithServerAllowedOrigins 设置允许的跨域来源验证函数
func WithServerAllowedOrigins(fn func(origin string) bool) ServerOption {
	return server.WithServerAllowedOrigins(fn)
}

// WithServerMaxMessageSize 设置最大消息大小（字节），0 不限制
func WithServerMaxMessageSize(size int) ServerOption { return server.WithServerMaxMessageSize(size) }

// ============================================================================
// 客户端选项函数
// ============================================================================

// WithClientCodec 设置编解码器，支持 "json"/"msgpack"/"protobuf"，默认 "json"
func WithClientCodec(name string) ClientOption { return client.WithClientCodec(name) }

// WithClientDialTimeout 设置连接建立超时，默认 30s
func WithClientDialTimeout(timeout time.Duration) ClientOption {
	return client.WithClientDialTimeout(timeout)
}

// WithClientAutoReconnect 设置是否启用自动重连
func WithClientAutoReconnect(enabled bool) ClientOption {
	return client.WithClientAutoReconnect(enabled)
}

// WithClientMaxReconnectAttempts 设置最大重连次数，默认 10
func WithClientMaxReconnectAttempts(n int) ClientOption {
	return client.WithClientMaxReconnectAttempts(n)
}

// WithClientHeartbeat 设置健康检查间隔，默认 15s，0 禁用
func WithClientHeartbeat(interval time.Duration) ClientOption {
	return client.WithClientHeartbeat(interval)
}
