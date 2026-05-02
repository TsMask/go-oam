package grpc

import "context"

// ServerHandler 服务端处理函数类型
// 参数：
//   - ctx: 上下文
//   - clientID: 客户端ID
//   - payload: 请求数据
//
// 返回：
//   - response: 响应数据
//   - error: 错误
type ServerHandler func(ctx context.Context, clientID string, payload []byte) ([]byte, error)

// ClientHandler 客户端处理函数类型（处理服务端发起的调用）
// 参数：
//   - ctx: 上下文
//   - payload: 请求数据
//
// 返回：
//   - response: 响应数据
//   - error: 错误
type ClientHandler func(ctx context.Context, payload []byte) ([]byte, error)

// serverHandlers 服务端处理器映射
type serverHandlers struct {
	handlers map[string]ServerHandler
}

func newServerHandlers() *serverHandlers {
	return &serverHandlers{
		handlers: make(map[string]ServerHandler),
	}
}

func (sh *serverHandlers) Handle(action string, handler ServerHandler) {
	sh.handlers[action] = handler
}

func (sh *serverHandlers) Get(action string) (ServerHandler, bool) {
	h, ok := sh.handlers[action]
	return h, ok
}

// clientHandlers 客户端处理器映射
type clientHandlers struct {
	handlers map[string]ClientHandler
	actions  []string // 注册的 actions 列表，用于连接时上报
}

func newClientHandlers() *clientHandlers {
	return &clientHandlers{
		handlers: make(map[string]ClientHandler),
		actions:  make([]string, 0),
	}
}

func (ch *clientHandlers) Handle(action string, handler ClientHandler) {
	ch.handlers[action] = handler
	ch.actions = append(ch.actions, action)
}

func (ch *clientHandlers) Get(action string) (ClientHandler, bool) {
	h, ok := ch.handlers[action]
	return h, ok
}

func (ch *clientHandlers) Actions() []string {
	return ch.actions
}
