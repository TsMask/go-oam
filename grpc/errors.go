package grpc

import "errors"

var (
	// ErrClientNotFound 客户端未找到
	ErrClientNotFound = errors.New("grpc: client not found")
	// ErrActionNotFound action 未注册
	ErrActionNotFound = errors.New("grpc: action not found")
	// ErrRequestTimeout 请求超时
	ErrRequestTimeout = errors.New("grpc: request timeout")
	// ErrStreamClosed 流已关闭
	ErrStreamClosed = errors.New("grpc: stream closed")
	// ErrServerUnavailable 服务端不可达
	ErrServerUnavailable = errors.New("grpc: server unavailable")
	// ErrInvalidState 无效状态
	ErrInvalidState = errors.New("grpc: invalid state")
	// ErrCodecError 编解码错误
	ErrCodecError = errors.New("grpc: codec error")
	// ErrConnectionLost 连接丢失
	ErrConnectionLost = errors.New("grpc: connection lost")
	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("grpc: client closed")
	// ErrSendFull 发送通道满
	ErrSendFull = errors.New("grpc: send channel full")
	// ErrRateLimited 请求被限流
	ErrRateLimited = errors.New("grpc: rate limited")
	// ErrBackoff 背压触发
	ErrBackoff = errors.New("grpc: backoff")
)
