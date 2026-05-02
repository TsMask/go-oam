package ws

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/core"
)

// 重连配置常量
const (
	// BaseReconnectDelay 基础重连延迟
	BaseReconnectDelay = 1 * time.Second
	// MaxReconnectDelay 最大重连延迟
	MaxReconnectDelay = 60 * time.Second
	// MaxReconnectAttempts 最大重连尝试次数
	MaxReconnectAttempts = 10
)

// 分片配置：256个分片，减少锁竞争
const (
	shardCount = 256            // 分片数量
	shardMask  = shardCount - 1 // 分片掩码，用于快速取模
)

// State 连接状态
type State int

// 连接状态常量
const (
	StateInit          State = iota // 初始状态
	StateConnecting                 // 连接中
	StateConnected                  // 已连接
	StateDisconnecting              // 断开中
	StateDisconnected               // 已断开
	StateReconnecting               // 重连中
	StateFailed                     // 连接失败
)

// String 返回状态字符串
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

// IsActive 检查是否活跃
func (s State) IsActive() bool {
	return s == StateConnected
}

// IsClosed 检查是否已关闭
func (s State) IsClosed() bool {
	return s == StateDisconnected || s == StateFailed
}

// CanConnect 检查是否可以连接
func (s State) CanConnect() bool {
	return s == StateInit || s == StateDisconnected || s == StateFailed
}

// pendingReq 待处理请求
type pendingReq struct {
	id      string
	ch      chan *Response
	timeout time.Time
	sent    atomic.Bool // 请求是否已发送
}

// ClientConfig 客户端配置
type ClientConfig struct {
	SendBufferSize     int           // 发送队列大小
	Workers            int           // Worker数量
	BatchSize          int           // 批量发送大小
	BatchTimeout       time.Duration // 批量超时
	RateLimit          float64       // 限流速率
	BackoffHigh        int           // 背压高水位
	BackoffLow         int           // 背压低水位
	DialTimeout        time.Duration // 连接超时
	ReadTimeout        time.Duration // 读取超时
	WriteTimeout       time.Duration // 写入超时
	MaxPendingRequests int           // 最大pending请求数
	EnableMetrics      bool          // 是否启用指标收集
	AutoReconnect      bool          // 是否启用自动重连
	ReconnectDelay     time.Duration // 重连延迟
}

// Client 高性能WebSocket客户端
type Client struct {
	url    string
	conn   *websocket.Conn
	codec  codec.Codec
	state  atomic.Int32
	sendCh chan []byte

	onState func(State)
	onError func(error)
	onResp  func(*Response)

	closed atomic.Bool

	// 分片pending请求表
	shards [shardCount]struct {
		mu      sync.Mutex
		pending map[string]*pendingReq
	}

	pool        *core.Pool
	batch       *core.BatchScheduler
	rateLimit   *core.AtomicRateLimiter
	backoff     *core.AdaptiveBackoff

	cfg ClientConfig

	metrics *ClientMetrics

	pendingReqPool sync.Pool
	cleanupIdx     atomic.Int64
}

// ClientMetrics 客户端性能指标
type ClientMetrics struct {
	SendQueueSize    atomic.Int64
	RecvQueueSize    atomic.Int64
	ActiveRequests   atomic.Int64
	FailedSends      atomic.Int64
	BackpressureHits atomic.Int64
	RateLimitDrops   atomic.Int64
	RequestLatency   atomic.Int64
	PoolHits         atomic.Int64
	ReconnectCount   atomic.Int64
}

// NewClient 创建WebSocket客户端
func NewClient(url string, codec codec.Codec, opts ...ClientOption) *Client {
	cfg := ClientConfig{
		SendBufferSize:     1000,
		Workers:            1,
		BatchSize:          100,
		BatchTimeout:       5 * time.Millisecond,
		RateLimit:          10000,
		BackoffHigh:        8000,
		BackoffLow:         2000,
		DialTimeout:        30 * time.Second,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxPendingRequests: 10000,
		ReconnectDelay:     3 * time.Second,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	c := &Client{
		url:            url,
		codec:          codec,
		sendCh:         make(chan []byte, cfg.SendBufferSize),
		cfg:            cfg,
		pool:           core.NewPool(),
		rateLimit:      core.NewAtomicRateLimiter(cfg.RateLimit, cfg.MaxPendingRequests),
		backoff:        core.NewAdaptiveBackoff(),
		metrics:        &ClientMetrics{},
	}

	for i := range c.shards {
		c.shards[i].pending = make(map[string]*pendingReq)
	}

	c.pendingReqPool.New = func() any {
		return &pendingReq{}
	}

	if cfg.BatchSize > 0 && cfg.BatchTimeout > 0 {
		c.batch = core.NewBatchScheduler(cfg.BatchSize, cfg.BatchTimeout, c.flushBatch)
	}

	return c
}

// Connect 建立WebSocket连接
func (c *Client) Connect(ctx context.Context) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	c.state.Store(int32(StateConnecting))

	dialer := websocket.Dialer{
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		HandshakeTimeout: c.cfg.DialTimeout,
	}

	if c.cfg.DialTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.DialTimeout)
		defer cancel()
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		c.state.Store(int32(StateFailed))
		return err
	}

	c.conn = conn
	c.state.Store(int32(StateConnected))

	if c.cfg.ReadTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
	}

	if c.onState != nil {
		c.onState(StateConnected)
	}

	go c.readLoop()

	if c.batch != nil {
		go c.batch.Run()
	} else {
		go c.writeLoop()
	}

	go c.cleanupPendingRequests()

	return nil
}

// cleanupPendingRequests 后台清理超时的pending请求
// 使用分片轮询优化：每 1 秒只清理一个分片，256 秒完成一轮
func (c *Client) cleanupPendingRequests() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if c.closed.Load() {
			return
		}

		idx := c.cleanupIdx.Add(1) % shardCount
		now := time.Now()

		c.shards[idx].mu.Lock()
		for id, pr := range c.shards[idx].pending {
			if now.After(pr.timeout) {
				var code int32
				var msg string
				if pr.sent.Load() {
					// 已发送但未收到响应 - 网络问题
					code = 503
					msg = "connection lost, request abandoned"
				} else {
					// 未发送就超时 - 请求超时
					code = 408
					msg = "request timeout"
				}
				select {
				case pr.ch <- &Response{
					ID:   id,
					Code: code,
					Msg:  msg,
				}:
				default:
				}
				delete(c.shards[idx].pending, id)
			}
		}
		c.shards[idx].mu.Unlock()
	}
}

// Close 关闭WebSocket连接
func (c *Client) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}

	if c.conn != nil {
		c.conn.Close()
	}

	if c.batch != nil {
		c.batch.Close()
	}

	c.state.Store(int32(StateDisconnected))

	if c.onState != nil {
		c.onState(StateDisconnected)
	}
}

// Send 同步发送，等待响应
func (c *Client) Send(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Microseconds()
		c.metrics.RequestLatency.Add(int64(latency))
	}()

	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	if c.state.Load() != int32(StateConnected) {
		return nil, ErrInvalidState
	}

	if c.rateLimit != nil && !c.rateLimit.Allow() {
		c.metrics.RateLimitDrops.Add(1)
		return nil, ErrRateLimited
	}

	id := req.ID
	if id == "" {
		id = gonanoid.Must(21)
		req.ID = id
	}

	if c.cfg.MaxPendingRequests > 0 {
		activeReqs := c.metrics.ActiveRequests.Load()
		if activeReqs >= int64(c.cfg.MaxPendingRequests) {
			c.metrics.RateLimitDrops.Add(1)
			return nil, ErrTooManyRequests
		}
	}

	data, err := c.codec.MarshalRequest(req)
	if err != nil {
		return nil, err
	}

	idx := int(hashString(id) & shardMask)

	pr := c.getPendingReq()
	pr.id = id
	pr.timeout = time.Now().Add(c.cfg.DialTimeout)

	c.shards[idx].mu.Lock()
	c.shards[idx].pending[id] = pr
	c.shards[idx].mu.Unlock()

	defer func() {
		c.shards[idx].mu.Lock()
		delete(c.shards[idx].pending, id)
		c.shards[idx].mu.Unlock()
		c.putPendingReq(pr)
	}()

	if c.batch != nil {
		queueLen := len(c.sendCh)
		c.backoff.Record(queueLen)

		if c.backoff.ShouldBackoff() {
			c.metrics.BackpressureHits.Add(1)
			return nil, ErrBackoff
		}
		if queueLen >= c.cfg.BackoffHigh {
			c.metrics.BackpressureHits.Add(1)
			return nil, ErrBackoffQueueFull
		}
		if !c.batch.Submit(data) {
			c.metrics.BackpressureHits.Add(1)
			return nil, ErrBackoffBatchFull
		}
		pr.sent.Store(true)
	} else {
		select {
		case c.sendCh <- data:
			pr.sent.Store(true)
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.cfg.DialTimeout):
			return nil, ErrTimeout
		}
	}

	c.metrics.ActiveRequests.Add(1)
	defer c.metrics.ActiveRequests.Add(-1)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-pr.ch:
		return resp, nil
	case <-time.After(c.cfg.DialTimeout):
		return nil, ErrTimeout
	}
}

// SendAsync 异步发送，回调
func (c *Client) SendAsync(ctx context.Context, req *Request, callback func(*Response, error)) {
	go func() {
		resp, err := c.Send(ctx, req)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

// SendRaw 裸数据发送（不等待响应）
func (c *Client) SendRaw(data []byte) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	if c.backoff.ShouldBackoff() {
		c.metrics.BackpressureHits.Add(1)
		return ErrBackoff
	}

	c.metrics.SendQueueSize.Store(int64(len(c.sendCh)))

	select {
	case c.sendCh <- data:
		return nil
	default:
		c.metrics.FailedSends.Add(1)
		return ErrSendFull
	}
}

func (c *Client) getPendingReq() *pendingReq {
	pr := c.pendingReqPool.Get().(*pendingReq)
	c.metrics.PoolHits.Add(1)
	return pr
}

func (c *Client) putPendingReq(pr *pendingReq) {
	pr.id = ""
	pr.timeout = time.Time{}
	select {
	case <-pr.ch:
	default:
	}
	c.pendingReqPool.Put(pr)
}

func (c *Client) readLoop() {
	defer c.Close()

	for {
		if c.cfg.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			c.onError(ErrConnectionLost)

			if c.cfg.AutoReconnect && !c.closed.Load() {
				c.metrics.ReconnectCount.Add(1)
				go c.reconnect()
			}
			return
		}

		if msgType == websocket.PingMessage {
			c.conn.WriteMessage(websocket.PingMessage, nil)
			continue
		}

		resp, err := c.codec.UnmarshalResponse(data)
		if err != nil {
			c.onError(ErrCodecError)
			continue
		}

		if resp != nil && resp.ID != "" {
			c.deliverResponse(resp)
		}

		if c.onResp != nil {
			c.onResp(resp)
		}
	}
}

// reconnect 自动重连，带指数退避和抖动
func (c *Client) reconnect() {
	for attempt := range MaxReconnectAttempts {
		if c.closed.Load() {
			return
		}

		delay := BaseReconnectDelay * time.Duration(1<<uint(attempt))
		if delay > MaxReconnectDelay {
			delay = MaxReconnectDelay
		}

		jitter := time.Duration(rand.Int63n(int64(delay / 4)))
		delay = delay - jitter/2 + jitter

		time.Sleep(delay)

		c.state.Store(int32(StateConnecting))
		if c.onState != nil {
			c.onState(StateConnecting)
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.cfg.DialTimeout)
		if err := c.Connect(ctx); err != nil {
			cancel()
			c.state.Store(int32(StateFailed))
			continue
		}
		cancel()

		return
	}

	c.state.Store(int32(StateFailed))
	if c.onError != nil {
		c.onError(ErrConnectionLost)
	}
}

func (c *Client) deliverResponse(resp *Response) {
	idx := int(hashString(resp.ID) & shardMask)
	c.shards[idx].mu.Lock()
	pr, ok := c.shards[idx].pending[resp.ID]
	c.shards[idx].mu.Unlock()

	if ok && pr != nil {
		select {
		case pr.ch <- resp:
		default:
		}
	}
}

func (c *Client) writeLoop() {
	for data := range c.sendCh {
		c.metrics.SendQueueSize.Store(int64(len(c.sendCh)))

		if c.cfg.WriteTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
		}

		if err := c.conn.WriteMessage(c.codec.MessageType(), data); err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			c.metrics.FailedSends.Add(1)
			continue
		}
	}
}

func (c *Client) flushBatch(batch [][]byte) {
	for _, data := range batch {
		if c.cfg.WriteTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
		}

		if err := c.conn.WriteMessage(c.codec.MessageType(), data); err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			c.metrics.FailedSends.Add(1)
			continue
		}
	}
}

// State 获取当前连接状态
func (c *Client) State() State {
	return State(c.state.Load())
}

// OnState 注册状态变更回调
func (c *Client) OnState(fn func(State)) {
	c.onState = fn
}

// OnError 注册错误处理回调
func (c *Client) OnError(fn func(error)) {
	c.onError = fn
}

// OnReceive 注册响应接收回调
func (c *Client) OnReceive(fn func(*Response)) {
	c.onResp = fn
}

// Metrics 获取性能指标
func (c *Client) Metrics() *ClientMetrics {
	return c.metrics
}

// RateLimiter 获取限流器
func (c *Client) RateLimiter() *core.AtomicRateLimiter {
	return c.rateLimit
}

// BackoffController 获取背压控制器（包含指标）
func (c *Client) BackoffController() *core.AdaptiveBackoff {
	return c.backoff
}

func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
