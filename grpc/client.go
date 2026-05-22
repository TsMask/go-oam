package grpc

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
)

// Client gRPC 客户端
type Client struct {
	endpoint string

	// 调用服务端的连接
	callConn *Conn
	// 接收服务端推送的连接
	pushConn *Conn

	state atomic.Int32

	handlers *clientHandlers

	closed atomic.Bool

	cfg ClientConfig

	onState func(State)
	onError func(error)
}

// State 连接状态
type State int

const (
	StateInit State = iota
	StateConnecting
	StateConnected
	StateDisconnecting
	StateDisconnected
	StateReconnecting
	StateFailed
)

func (s State) String() string {
	switch s {
	case StateInit:
		return "init"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateReconnecting:
		return "reconnecting"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

func (s State) IsActive() bool {
	return s == StateConnected
}

func (s State) IsClosed() bool {
	return s == StateDisconnected || s == StateFailed
}

// NewClient 创建 gRPC 客户端
func NewClient(endpoint string, opts ...ClientOption) *Client {
	cfg := ClientConfig{
		SendBufferSize:     1000,
		Workers:           1,
		RateLimit:         10000,
		BackoffHigh:       8000,
		BackoffLow:        2000,
		DialTimeout:       30 * time.Second,
		MaxPendingRequests: 10000,
		AutoReconnect:     false,
		ReconnectDelay:    3 * time.Second,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	c := &Client{
		endpoint: endpoint,
		handlers: newClientHandlers(),
		cfg:      cfg,
	}

	return c
}

// Handle 注册处理函数（处理服务端发起的调用）
func (c *Client) Handle(action string, handler ClientHandler) {
	c.handlers.Handle(action, handler)
}

// Connect 建立连接
func (c *Client) Connect(ctx context.Context) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	c.state.Store(int32(StateConnecting))

	// 创建调用服务端的连接
	c.callConn = newConn(c.cfg.SendBufferSize)
	callStream, err := newCallStream(ctx, c.endpoint, c.callConn)
	if err != nil {
		c.state.Store(int32(StateFailed))
		return err
	}

	// 创建接收服务端推送的连接
	c.pushConn = newConn(c.cfg.SendBufferSize)
	pushStream, err := newPushStream(ctx, c.endpoint, c.pushConn)
	if err != nil {
		c.state.Store(int32(StateFailed))
		return err
	}

	c.state.Store(int32(StateConnected))

	// 启动调用流读写循环
	go callStream.readLoop()
	go callStream.writeLoop()

	// 启动推送流读写循环
	go pushStream.readLoop()
	go pushStream.writeLoop()

	return nil
}

// Call 调用服务端
func (c *Client) Call(ctx context.Context, action string, data []byte) ([]byte, error) {
	return c.callConn.Call(ctx, action, data)
}

// CallAsync 异步调用服务端
func (c *Client) CallAsync(ctx context.Context, action string, data []byte, callback func([]byte, error)) {
	c.callConn.CallAsync(ctx, action, data, callback)
}

// Close 关闭客户端
func (c *Client) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}

	if c.callConn != nil {
		c.callConn.Close()
	}
	if c.pushConn != nil {
		c.pushConn.Close()
	}

	c.state.Store(int32(StateDisconnected))
}

// State 获取当前状态
func (c *Client) State() State {
	return State(c.state.Load())
}

// OnState 注册状态变更回调
func (c *Client) OnState(fn func(State)) {
	c.onState = fn
}

// OnError 注册错误回调
func (c *Client) OnError(fn func(error)) {
	c.onError = fn
}

// NumGoroutine 返回当前goroutine数量
func (c *Client) NumGoroutine() int {
	return runtime.NumGoroutine()
}
