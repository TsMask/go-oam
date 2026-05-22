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
	// ErrTimeout 操作超时
	ErrTimeout = errors.New("timeout")
	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("client closed")
	// ErrConnectionLost 连接丢失（读/写失败）
	ErrConnectionLost = errors.New("connection lost")
	// ErrRateLimited 请求被限流
	ErrRateLimited = errors.New("rate limited")
	// ErrInvalidState 无效状态（状态机错误）
	ErrInvalidState = errors.New("invalid state")
)

// ============================================================================
// 类型定义
// ============================================================================

// State 连接状态
type State int

const (
	StateInit          State = iota // 初始状态
	StateConnecting                 // 连接中
	StateConnected                  // 已连接
	StateDisconnecting              // 断开中
	StateDisconnected               // 已断开
	StateReconnecting               // 重连中
	StateFailed                     // 连接失败
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

// pendingReq 待处理请求
type pendingReq struct {
	ch      chan *types.Response
	timeout time.Time
}

// ClientConfig 客户端配置
type ClientConfig struct {
	SendBufferSize       int           // 发送队列大小，默认 1000
	DialTimeout          time.Duration // 连接建立超时，默认 30s
	RequestTimeout       time.Duration // 请求响应超时，默认 60s
	MaxPendingRequests   int           // 最大并发 pending 请求数，默认 10000
	AutoReconnect        bool          // 自动重连，默认 false
	MaxReconnectAttempts int           // 最大重连次数，默认 10
}

// Client WebSocket 客户端
// 支持请求-响应模式、自动重连
type Client struct {
	url    string
	conn   *websocket.Conn
	codec  codec.Codec
	state  atomic.Int32
	sendCh chan []byte
	done   chan struct{}
	doneMu sync.Mutex // 保护 done channel 替换

	onState func(State)
	onError func(error)
	onResp  func(*types.Response)

	closed atomic.Bool

	pending   map[string]*pendingReq
	pendingMu sync.RWMutex

	cleanupDone chan struct{} // cleanup goroutine 退出信号

	cfg ClientConfig
}

// NewClient 创建 WebSocket 客户端
func NewClient(url string, codec codec.Codec, opts ...ClientOption) *Client {
	cfg := ClientConfig{
		SendBufferSize:       1000,
		DialTimeout:          30 * time.Second,
		RequestTimeout:       60 * time.Second,
		MaxPendingRequests:   10000,
		MaxReconnectAttempts: 10,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Client{
		url:         url,
		codec:       codec,
		sendCh:      make(chan []byte, cfg.SendBufferSize),
		done:        make(chan struct{}),
		pending:     make(map[string]*pendingReq),
		cleanupDone: make(chan struct{}),
		cfg:         cfg,
	}
}

// Connect 建立 WebSocket 连接
func (c *Client) Connect(ctx context.Context) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	c.state.Store(int32(StateConnecting))

	dialCtx, dialCancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, c.url, nil)
	if err != nil {
		c.state.Store(int32(StateFailed))
		return err
	}

	// 安全替换 done channel
	c.doneMu.Lock()
	c.done = make(chan struct{})
	c.doneMu.Unlock()

	c.conn = conn
	c.state.Store(int32(StateConnected))

	if c.onState != nil {
		c.onState(StateConnected)
	}

	go c.readLoop()
	go c.writeLoop()
	go c.cleanupPendingRequests()

	return nil
}

// Close 关闭 WebSocket 连接
func (c *Client) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}

	c.doneMu.Lock()
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.doneMu.Unlock()

	close(c.cleanupDone)

	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "client closed")
	}
	c.state.Store(int32(StateDisconnected))
	if c.onState != nil {
		c.onState(StateDisconnected)
	}
}

// Send 同步发送请求并等待响应
func (c *Client) Send(ctx context.Context, req *types.Request) (*types.Response, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}
	if c.state.Load() != int32(StateConnected) {
		return nil, ErrInvalidState
	}

	id := req.ID
	if id == "" {
		id = generate.String(21)
		req.ID = id
	}

	// 检查 pending 数量
	c.pendingMu.RLock()
	pendingCount := len(c.pending)
	c.pendingMu.RUnlock()
	if c.cfg.MaxPendingRequests > 0 && pendingCount >= c.cfg.MaxPendingRequests {
		return nil, ErrRateLimited
	}

	data, err := c.codec.MarshalRequest(req)
	if err != nil {
		return nil, err
	}

	// 注册 pending
	pr := &pendingReq{
		ch:      make(chan *types.Response, 1),
		timeout: time.Now().Add(c.cfg.RequestTimeout),
	}

	c.pendingMu.Lock()
	c.pending[id] = pr
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	// 发送消息
	sendTimer := time.NewTimer(c.cfg.RequestTimeout)
	select {
	case c.sendCh <- data:
		sendTimer.Stop()
	case <-ctx.Done():
		sendTimer.Stop()
		return nil, ctx.Err()
	case <-sendTimer.C:
		return nil, ErrTimeout
	}

	// 等待响应
	respTimer := time.NewTimer(c.cfg.RequestTimeout)
	defer respTimer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-pr.ch:
		return resp, nil
	case <-respTimer.C:
		return nil, ErrTimeout
	}
}

// SendAsync 异步发送，回调处理结果
func (c *Client) SendAsync(ctx context.Context, req *types.Request, callback func(*types.Response, error)) {
	go func() {
		resp, err := c.Send(ctx, req)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

// SendWithTimeout 发送带超时的请求
func (c *Client) SendWithTimeout(req *types.Request, timeout time.Duration) (*types.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Send(ctx, req)
}

// OnState 设置状态变更回调
func (c *Client) OnState(fn func(State)) { c.onState = fn }

// OnError 设置错误回调
func (c *Client) OnError(fn func(error)) { c.onError = fn }

// OnReceive 设置响应回调（所有收到的响应都会触发，包括 Send 匹配的）
func (c *Client) OnReceive(fn func(*types.Response)) { c.onResp = fn }

// State 获取当前连接状态
func (c *Client) State() State { return State(c.state.Load()) }

// cleanupPendingRequests 后台清理超时的 pending 请求
func (c *Client) cleanupPendingRequests() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.cleanupDone:
			return
		case <-ticker.C:
			now := time.Now()
			c.pendingMu.Lock()
			for id, pr := range c.pending {
				if now.After(pr.timeout) {
					select {
					case pr.ch <- &types.Response{ID: id, Code: 408, Msg: "request timeout"}:
					default:
					}
					delete(c.pending, id)
				}
			}
			c.pendingMu.Unlock()
		}
	}
}

// readLoop 读取循环
func (c *Client) readLoop() {
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		c.doneMu.Lock()
		doneCh := c.done
		c.doneMu.Unlock()
		select {
		case <-doneCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if c.onError != nil {
				c.onError(ErrConnectionLost)
			}
			if c.cfg.AutoReconnect && !c.closed.Load() {
				go c.reconnect()
			}
			return
		}

		resp, err := c.codec.UnmarshalResponse(data)
		if err != nil {
			continue
		}

		// 分发响应到 pending
		if resp != nil && resp.ID != "" {
			c.pendingMu.RLock()
			pr, ok := c.pending[resp.ID]
			c.pendingMu.RUnlock()
			if ok {
				select {
				case pr.ch <- resp:
				default:
				}
			}
		}

		if c.onResp != nil {
			c.onResp(resp)
		}
	}
}

// reconnect 自动重连，指数退避 + 抖动
func (c *Client) reconnect() {
	// 确保旧的 done channel 被关闭
	c.doneMu.Lock()
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.doneMu.Unlock()

	// 等待旧 goroutine 退出
	time.Sleep(100 * time.Millisecond)

	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	for attempt := 0; attempt < c.cfg.MaxReconnectAttempts; attempt++ {
		if c.closed.Load() {
			return
		}

		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}
		// 抖动 ±12.5%
		jitter := time.Duration(rand.Int63n(int64(delay) / 4))
		delay = delay - delay/8 + jitter

		time.Sleep(delay)

		c.state.Store(int32(StateReconnecting))
		if c.onState != nil {
			c.onState(StateReconnecting)
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.cfg.DialTimeout)
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

// writeLoop 写入循环
func (c *Client) writeLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		c.doneMu.Lock()
		doneCh := c.done
		c.doneMu.Unlock()
		select {
		case <-doneCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	msgType := websocket.MessageType(c.codec.MessageType())

	for {
		select {
		case <-ctx.Done():
			// drain 剩余消息
			for {
				select {
				case data := <-c.sendCh:
					_ = c.conn.Write(context.Background(), msgType, data)
				default:
					return
				}
			}
		case data := <-c.sendCh:
			if err := c.conn.Write(ctx, msgType, data); err != nil {
				if c.onError != nil {
					c.onError(err)
				}
				return
			}
		}
	}
}
