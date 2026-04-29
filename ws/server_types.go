package ws

import (
	"errors"
)

// 错误定义
var (
	ErrMaxConns        = errors.New("max connections reached") // 最大连接数（server.go L379）
	ErrServerClosed    = errors.New("server closed")           // 服务端已关闭（server.go L373）
	ErrConnectionLost  = errors.New("connection lost")         // 连接断开（client.go L468）
	ErrCodecError      = errors.New("codec error")             // 编解码错误（client.go L483）
	ErrHandlerNotFound = errors.New("handler not found")       // 处理器不存在（server.go readLoop）
)
