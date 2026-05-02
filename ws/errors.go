package ws

import "errors"

// 错误定义
var (
	// ErrServerClosed 服务器已关闭（调用 Shutdown 后）
	ErrServerClosed = errors.New("server closed")
	// ErrConnClosed 连接已关闭
	ErrConnClosed = errors.New("connection closed")
	// ErrSendFull 发送通道满（背压）
	ErrSendFull = errors.New("send channel full")

	// 背压相关错误
	// ErrBackoff 背压应用（自适应背压控制器触发）
	ErrBackoff = errors.New("backoff: backpressure applied")
	// ErrBackoffQueueFull 发送队列满（本地队列积压）
	ErrBackoffQueueFull = errors.New("backoff: send queue full")
	// ErrBackoffBatchFull 批处理队列满（批量调度器满）
	ErrBackoffBatchFull = errors.New("backoff: batch queue full")

	// ErrRateLimited 请求被限流
	ErrRateLimited = errors.New("rate limited")
	// ErrTimeout 操作超时
	ErrTimeout = errors.New("timeout")
	// ErrInvalidState 无效状态（状态机错误）
	ErrInvalidState = errors.New("invalid state")
	// ErrTooManyRequests 请求过多（超出并发限制）
	ErrTooManyRequests = errors.New("too many requests")
	// ErrCodecError 编解码错误
	ErrCodecError = errors.New("codec error")
	// ErrConnectionLost 连接丢失（读/写失败）
	ErrConnectionLost = errors.New("connection lost")
	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("client closed")

	// 以下是服务端专用错误

	// ErrMaxConnections 最大连接数（超出 server MaxConns 限制）
	ErrMaxConnections = errors.New("max connections reached")
	// ErrHandlerNotFound 处理器不存在
	ErrHandlerNotFound = errors.New("handler not found")
)
