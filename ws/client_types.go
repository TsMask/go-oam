package ws

import (
	"errors"
)

// 错误定义
var (
	ErrSendFull        = errors.New("send channel full")         // 发送队列满（client.go L451, server.go L165）
	ErrRateLimited     = errors.New("rate limited")              // 限流触发（client.go L364）
	ErrTimeout         = errors.New("timeout")                   // 超时（client.go L410, 425）
	ErrBackoff         = errors.New("backpressure")              // 背压触发（client.go L402, 440）
	ErrClientClosed    = errors.New("client closed")             // 客户端已关闭（client.go L270, 354, 434）
	ErrInvalidState    = errors.New("invalid state")             // 连接状态无效（client.go L358）
	ErrTooManyRequests = errors.New("too many pending requests") // pending请求过多（client.go L425）
)
