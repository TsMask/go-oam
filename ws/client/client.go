package client

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/tsmask/go-oam/pkg/generate"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/types"
)

// ============================================================================
// 错误定义
// ============================================================================

var (
	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("client closed")
	// ErrConnectionLost 连接丢失（读/写失败）
	ErrConnectionLost = errors.New("connection lost")
	// ErrInvalidState 无效状态（状态机错误）
	ErrInvalidState = errors.New("invalid state")
)

// ============================================================================
// 类型定义
// ============================================================================

// State 连接状态
type State int

const (
	StateInit         State = iota // 初始状态
	StateConnecting                // 连接中
	StateConnected                 // 已连接
	StateDisconnected              // 已断开
	StateReconnecting              // 重连中
	StateFailed                    // 连接失败
)

func (s State) String() string {
	switch s {
	case StateInit:
		return "init"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
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

// Client WebSocket 客户端
// 发送消息后不等响应，通过 OnReceive 回调异步接收所有响应
//
// 生命周期管理采用双层 context：
//   - ctx（客户端级）：Close() 取消，停止一切活动
//   - connCtx（连接级）：每次 Connect 创建，连接丢失时取消，不影响客户端级
type Client struct {
	url   string
	codec codec.Codec
	cfg   clientConfig

	sendCh chan []byte

	// 客户端生命周期，Close() 取消
	ctx    context.Context
	cancel context.CancelFunc

	// 当前连接生命周期，每次 Connect 创建新的
	conn       *websocket.Conn
	connMu     sync.Mutex
	connCtx    context.Context
	connCancel context.CancelFunc

	state atomic.Int32

	onState   func(State)
	onError   func(error)
	onReceive func(*types.Response)
}

// NewClient 创建 WebSocket 客户端
func NewClient(url string, opts ...ClientOption) *Client {
	cfg := clientConfig{
		dialTimeout:          30 * time.Second,
		maxReconnectAttempts: 10,
		heartbeat:            15 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.codec == "" {
		cfg.codec = "json"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		url:    url,
		codec:  codec.NewCodec(cfg.codec),
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
		sendCh: make(chan []byte, 512),
	}
}

// ============================================================================
// 公开方法
// ============================================================================

// Connect 建立 WebSocket 连接
func (c *Client) Connect(ctx context.Context) error {
	if c.ctx.Err() != nil {
		return ErrClientClosed
	}

	c.state.Store(int32(StateConnecting))

	dialCtx, dialCancel := context.WithTimeout(ctx, c.cfg.dialTimeout)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, c.url, nil)
	if err != nil {
		c.state.Store(int32(StateFailed))
		return err
	}

	// 取消旧连接（如果存在），清理旧 goroutine
	c.closeConn()

	// 创建连接级 context，父级为客户端 ctx
	connCtx, connCancel := context.WithCancel(c.ctx)

	c.connMu.Lock()
	c.conn = conn
	c.connCtx = connCtx
	c.connCancel = connCancel
	c.connMu.Unlock()

	c.state.Store(int32(StateConnected))
	if c.onState != nil {
		c.onState(StateConnected)
	}

	go c.readLoop(conn, connCtx)
	go c.writeLoop(conn, connCtx)
	if c.cfg.heartbeat > 0 {
		go c.healthLoop(conn, connCtx)
	}

	return nil
}

// Close 关闭客户端，停止所有活动
func (c *Client) Close() {
	if c.ctx.Err() != nil {
		return
	}
	c.cancel()
	c.closeConn()
	c.state.Store(int32(StateDisconnected))
	if c.onState != nil {
		c.onState(StateDisconnected)
	}
}

// Send 发送请求（非阻塞，不等响应）
// 响应通过 OnReceive 回调异步获取，按 resp.ID 匹配请求
func (c *Client) Send(req *types.Request) error {
	if c.ctx.Err() != nil {
		return ErrClientClosed
	}
	if c.state.Load() != int32(StateConnected) {
		return ErrInvalidState
	}

	// 不修改入参，内部生成 ID
	id := req.ID
	if id == "" {
		id = generate.String(21)
	}

	data, err := c.codec.MarshalRequest(&types.Request{
		ID:     id,
		Action: req.Action,
		Data:   req.Data,
	})
	if err != nil {
		return err
	}

	select {
	case c.sendCh <- data:
		return nil
	case <-c.ctx.Done():
		return ErrClientClosed
	}
}

// OnState 设置状态变更回调
func (c *Client) OnState(fn func(State)) { c.onState = fn }

// OnError 设置错误回调
func (c *Client) OnError(fn func(error)) { c.onError = fn }

// OnReceive 设置响应回调（所有收到的响应都会触发）
func (c *Client) OnReceive(fn func(*types.Response)) { c.onReceive = fn }

// State 获取当前连接状态
func (c *Client) State() State { return State(c.state.Load()) }

// ============================================================================
// 内部方法
// ============================================================================

// closeConn 取消当前连接级 context 并关闭底层连接
func (c *Client) closeConn() {
	c.connMu.Lock()
	if c.connCancel != nil {
		c.connCancel()
	}
	conn := c.conn
	c.conn = nil
	c.connCtx = nil
	c.connCancel = nil
	c.connMu.Unlock()

	if conn != nil {
		conn.CloseNow()
	}
}

// onConnectionLost 连接丢失处理
func (c *Client) onConnectionLost() {
	c.closeConn()

	// 主动关闭（Close() 已取消 ctx），不触发错误回调
	if c.ctx.Err() != nil {
		return
	}

	if c.onError != nil {
		c.onError(ErrConnectionLost)
	}

	if c.cfg.autoReconnect {
		go c.reconnect()
	} else {
		c.state.Store(int32(StateFailed))
	}
}

// readLoop 读取循环，参数为当前连接和对应 context
// 自动检测响应编码：binary 用配置的编码器，text 用 JSON 兜底
func (c *Client) readLoop(conn *websocket.Conn, ctx context.Context) {
	defer c.onConnectionLost()

	jsonCodec := codec.JSON()

	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		// 根据消息类型选择解码器
		var respCodec codec.Codec
		switch msgType {
		case websocket.MessageText:
			respCodec = jsonCodec
		default:
			respCodec = c.codec
		}

		resp, err := respCodec.UnmarshalResponse(data)
		if err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			continue
		}

		if c.onReceive != nil {
			c.onReceive(resp)
		}
	}
}

// writeLoop 写入循环
func (c *Client) writeLoop(conn *websocket.Conn, ctx context.Context) {
	msgType := websocket.MessageType(c.codec.MessageType())

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-c.sendCh:
			if err := conn.Write(ctx, msgType, data); err != nil {
				return
			}
		}
	}
}

// healthLoop 健康检查，定期 Ping 保持连接活跃
// coder/websocket 的 Ping 可与 Read 并发调用
func (c *Client) healthLoop(conn *websocket.Conn, ctx context.Context) {
	ticker := time.NewTicker(c.cfg.heartbeat)
	defer ticker.Stop()

	fails := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(context.Background(), c.cfg.heartbeat)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				fails++
				if fails >= 3 {
					c.closeConn()
					return
				}
				continue
			}
			fails = 0
		}
	}
}

// reconnect 自动重连，指数退避 + 抖动
func (c *Client) reconnect() {
	baseDelay := 500 * time.Millisecond
	maxDelay := 60 * time.Second

	for attempt := 0; attempt < c.cfg.maxReconnectAttempts; attempt++ {
		if c.ctx.Err() != nil {
			return
		}

		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}
		jitter := time.Duration(rand.Int63n(int64(delay) / 2))
		delay = delay/2 + jitter

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(delay):
		}

		c.state.Store(int32(StateReconnecting))
		if c.onState != nil {
			c.onState(StateReconnecting)
		}

		ctx, cancel := context.WithTimeout(c.ctx, c.cfg.dialTimeout)
		err := c.Connect(ctx)
		cancel()

		if err != nil {
			c.state.Store(int32(StateFailed))
			continue
		}
		return
	}

	c.state.Store(int32(StateFailed))
	if c.onError != nil {
		c.onError(ErrConnectionLost)
	}
}
