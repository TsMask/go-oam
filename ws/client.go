package ws

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/core"
	"github.com/tsmask/go-oam/ws/types"
)

// 分片配置：256个分片，减少锁竞争
const (
	shardCount = 256            // 分片数量
	shardMask  = shardCount - 1 // 分片掩码，用于快速取模
)

// ClientConfig 客户端配置
// 用于配置WebSocket客户端的各种参数
type ClientConfig struct {
	SendBufferSize     int           // 发送队列大小
	Workers            int           // Worker数量（用于批量发送）
	BatchSize          int           // 批量发送大小
	BatchTimeout       time.Duration // 批量超时
	RateLimit          float64       // 限流速率（请求/秒）
	BackoffHigh        int           // 背压高水位（队列大小）
	BackoffLow         int           // 背压低水位（队列大小）
	DialTimeout        time.Duration // 连接超时
	ReadTimeout        time.Duration // 读取超时
	WriteTimeout       time.Duration // 写入超时
	MaxPendingRequests int           // 最大pending请求数
	EnableMetrics      bool          // 是否启用指标收集
	AutoReconnect      bool          // 是否启用自动重连
	ReconnectDelay     time.Duration // 重连延迟
}

// pendingReq 待处理请求
// 用于实现请求-响应模型
type pendingReq struct {
	id      string               // 请求ID
	ch      chan *types.Response // 响应channel
	timeout time.Time            // 超时时间
}

// Client 高性能WebSocket客户端
// 特性：
//   - 支持同步请求-响应模型
//   - 内置限流和背压控制
//   - 支持批量发送减少网络开销
//   - 连接状态监控
//   - 可选自动重连
type Client struct {
	url    string          // 服务器地址
	conn   *websocket.Conn // WebSocket连接
	codec  codec.Codec     // 编解码器
	state  atomic.Int32    // 连接状态
	sendCh chan []byte     // 发送队列

	onState func(State)           // 状态变更回调
	onError func(error)           // 错误处理回调
	onResp  func(*types.Response) // 响应接收回调

	closed atomic.Bool // 关闭标记

	// 分片pending请求表，减少锁竞争
	shards [shardCount]struct {
		mu      sync.Mutex
		pending map[string]*pendingReq
	}

	pool           *core.Pool              // 内存对象池
	batch          *core.BatchScheduler    // 批量调度器
	rateLimit      *core.AtomicRateLimiter // 限流器（原子操作，无锁）
	backoff        *core.AdaptiveBackoff   // 背压控制器（自适应）
	backoffMetrics *core.BackoffMetrics    // 背压指标

	cfg ClientConfig // 配置

	metrics *ClientMetrics // 性能指标

	pendingReqPool sync.Pool // pending请求对象池
}

// ClientMetrics 客户端性能指标
// 用于监控客户端运行状态
type ClientMetrics struct {
	SendQueueSize    atomic.Int64 // 发送队列大小
	RecvQueueSize    atomic.Int64 // 接收队列大小
	ActiveRequests   atomic.Int64 // 活跃请求数
	FailedSends      atomic.Int64 // 发送失败次数
	BackpressureHits atomic.Int64 // 背压触发次数
	RateLimitDrops   atomic.Int64 // 限流丢弃次数
	RequestLatency   atomic.Int64 // 请求延迟（微秒）
	PoolHits         atomic.Int64 // 对象池命中次数
	ReconnectCount   atomic.Int64 // 重连次数
}

// NewClient 创建WebSocket客户端
// 参数：
//
//	url: 服务器地址，如 "ws://localhost:8080"
//	codec: 编解码器，如 ws.NewJSONCodec()
//	opts: 配置选项
//
// 返回：客户端实例
func NewClient(url string, codec codec.Codec, opts ...ClientOption) *Client {
	// 默认配置
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

	// 应用配置选项
	for _, opt := range opts {
		opt(&cfg)
	}

	// 创建客户端
	c := &Client{
		url:    url,
		codec:  codec,
		sendCh: make(chan []byte, cfg.SendBufferSize),
		cfg:    cfg,
		pool:   core.NewPool(),
		// 使用高性能组件：原子限流器 + 自适应背压
		rateLimit:      core.NewAtomicRateLimiter(cfg.RateLimit, cfg.MaxPendingRequests),
		backoff:        core.NewAdaptiveBackoff(),
		backoffMetrics: core.NewBackoffMetrics(),
		metrics:        &ClientMetrics{},
	}

	// 初始化分片
	for i := range c.shards {
		c.shards[i].pending = make(map[string]*pendingReq)
	}

	// 初始化pending请求对象池
	c.pendingReqPool.New = func() any {
		return &pendingReq{}
	}

	// 启用批量发送
	if cfg.BatchSize > 0 && cfg.BatchTimeout > 0 {
		c.batch = core.NewBatchScheduler(cfg.BatchSize, cfg.BatchTimeout, c.flushBatch)
	}

	return c
}

// getPendingReq 从对象池获取pending请求
func (c *Client) getPendingReq() *pendingReq {
	pr := c.pendingReqPool.Get().(*pendingReq)
	c.metrics.PoolHits.Add(1)
	return pr
}

// putPendingReq 归还pending请求到对象池
func (c *Client) putPendingReq(pr *pendingReq) {
	pr.id = ""
	pr.timeout = time.Time{}
	// 清空channel
	select {
	case <-pr.ch:
	default:
	}
	c.pendingReqPool.Put(pr)
}

// Connect 建立WebSocket连接
// 参数：ctx 上下文，用于超时控制
// 返回：连接错误
func (c *Client) Connect(ctx context.Context) error {
	// 检查是否已关闭
	if c.closed.Load() {
		return ErrClientClosed
	}

	c.state.Store(int32(StateConnecting))

	// 创建Dialer
	dialer := websocket.Dialer{
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		HandshakeTimeout: c.cfg.DialTimeout,
	}

	// 应用连接超时
	if c.cfg.DialTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.DialTimeout)
		defer cancel()
	}

	// 建立连接
	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		c.state.Store(int32(StateFailed))
		return err
	}

	c.conn = conn
	c.state.Store(int32(StateConnected))

	// 设置连接读取超时（用于心跳检测）
	if c.cfg.ReadTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
	}

	// 触发状态回调
	if c.onState != nil {
		c.onState(StateConnected)
	}

	// 启动读写循环
	go c.readLoop()

	if c.batch != nil {
		go c.batch.Run()
	} else {
		go c.writeLoop()
	}

	// 启动pending请求超时清理
	go c.cleanupPendingRequests()

	return nil
}

// cleanupPendingRequests 后台清理超时的pending请求
func (c *Client) cleanupPendingRequests() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if c.closed.Load() {
			return
		}

		now := time.Now()
		for i := range c.shards {
			c.shards[i].mu.Lock()
			for id, pr := range c.shards[i].pending {
				if now.After(pr.timeout) {
					select {
					case pr.ch <- &types.Response{
						ID:   id,
						Code: 413,
						Msg:  ErrTimeout.Error(),
					}:
					default:
					}
					delete(c.shards[i].pending, id)
				}
			}
			c.shards[i].mu.Unlock()
		}
	}
}

// Close 关闭WebSocket连接
// 安全关闭：停止接收新请求，等待现有请求完成
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

// Send 发送同步请求并等待响应
// 参数：
//
//	ctx: 上下文，用于超时和取消
//	req: 请求消息
//
// 返回：响应消息和错误
func (c *Client) Send(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()
	defer func() {
		// 记录延迟指标
		latency := time.Since(start).Microseconds()
		c.metrics.RequestLatency.Add(int64(latency))
	}()

	// 检查状态
	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	if c.state.Load() != int32(StateConnected) {
		return nil, ErrInvalidState
	}

	// 限流检查（使用原子操作，无锁）
	if c.rateLimit != nil && !c.rateLimit.Allow() {
		c.metrics.RateLimitDrops.Add(1)
		return nil, ErrRateLimited
	}

	// 生成请求ID（Nanoid）
	// 如果用户已经设置了ID，则使用用户的ID；否则自动生成Nanoid
	// Nanoid: 默认21字符，URL-safe
	id := req.ID
	if id == "" {
		id = gonanoid.Must(21)
		req.ID = id
	}

	// 检查最大pending请求数限制
	if c.cfg.MaxPendingRequests > 0 {
		activeReqs := c.metrics.ActiveRequests.Load()
		if activeReqs >= int64(c.cfg.MaxPendingRequests) {
			c.metrics.RateLimitDrops.Add(1)
			return nil, ErrTooManyRequests
		}
	}

	// 序列化请求
	data, err := c.codec.MarshalRequest(req)
	if err != nil {
		return nil, err
	}

	// 获取分片
	idx := int(hashString(id) & shardMask)

	// 创建pending请求
	pr := c.getPendingReq()
	pr.id = id
	pr.timeout = time.Now().Add(c.cfg.DialTimeout)

	// 注册pending请求
	c.shards[idx].mu.Lock()
	c.shards[idx].pending[id] = pr
	c.shards[idx].mu.Unlock()

	// 确保清理
	defer func() {
		c.shards[idx].mu.Lock()
		delete(c.shards[idx].pending, id)
		c.shards[idx].mu.Unlock()
		c.putPendingReq(pr)
	}()

	// 发送数据
	if c.batch != nil {
		// 记录背压指标
		queueLen := len(c.sendCh)
		c.backoff.Record(queueLen)

		// 使用背压高水位进行限制
		if c.backoff.ShouldBackoff() || queueLen >= c.cfg.BackoffHigh {
			c.metrics.BackpressureHits.Add(1)
			c.backoffMetrics.RecordBackoff()
			return nil, ErrBackoff
		}
		if !c.batch.Submit(data) {
			c.metrics.BackpressureHits.Add(1)
			c.backoffMetrics.RecordBackoff()
			return nil, ErrBackoff
		}
	} else {
		select {
		case c.sendCh <- data:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.cfg.DialTimeout):
			return nil, ErrTimeout
		}
	}

	// 增加活跃请求计数
	c.metrics.ActiveRequests.Add(1)
	defer c.metrics.ActiveRequests.Add(-1)

	// 等待响应
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-pr.ch:
		return resp, nil
	case <-time.After(c.cfg.DialTimeout):
		return nil, ErrTimeout
	}
}

// SendAsync 发送异步请求
// 使用回调函数处理响应，不阻塞当前goroutine
// 参数：
//
//	ctx: 上下文，用于超时控制
//	req: 请求消息
//	callback: 回调函数，当收到响应或超时时调用
func (c *Client) SendAsync(ctx context.Context, req *Request, callback func(*Response, error)) {
	go func() {
		resp, err := c.Send(ctx, req)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

// SendWithTimeout 发送带超时的请求
// 参数：
//
//	req: 请求消息
//	timeout: 超时时间
//
// 返回：响应消息和错误
func (c *Client) SendWithTimeout(req *Request, timeout time.Duration) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Send(ctx, req)
}

// SendRaw 发送数据（异步）
// 参数：data 二进制数据
// 返回：发送错误
func (c *Client) SendRaw(data []byte) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	// 背压检查
	if c.backoff.ShouldBackoff() {
		c.metrics.BackpressureHits.Add(1)
		c.backoffMetrics.RecordBackoff()
		return ErrBackoff
	}

	// 更新队列大小指标
	c.metrics.SendQueueSize.Store(int64(len(c.sendCh)))

	select {
	case c.sendCh <- data:
		return nil
	default:
		c.metrics.FailedSends.Add(1)
		return ErrSendFull
	}
}

// readLoop 读取循环
// 在独立goroutine中运行，处理服务器消息
func (c *Client) readLoop() {
	defer c.Close()

	for {
		// 设置读取超时
		if c.cfg.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
		}

		// 读取消息
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			// 检查是否是连接断开
			c.onError(ErrConnectionLost)

			// 自动重连
			if c.cfg.AutoReconnect && !c.closed.Load() {
				c.metrics.ReconnectCount.Add(1)
				go c.reconnect()
			}
			return
		}

		// 处理Ping
		if msgType == websocket.PingMessage {
			c.conn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		// 反序列化响应
		resp, err := c.codec.UnmarshalResponse(data)
		if err != nil {
			// 记录编解码错误
			c.onError(ErrCodecError)
			continue
		}

		// 分发响应
		if resp != nil && resp.ID != "" {
			c.deliverResponse(resp)
		}

		// 触发响应回调
		if c.onResp != nil {
			c.onResp(resp)
		}
	}
}

// reconnect 自动重连
func (c *Client) reconnect() {
	for i := 0; i < 5; i++ {
		if c.closed.Load() {
			return
		}

		time.Sleep(c.cfg.ReconnectDelay)

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

// deliverResponse 分发响应到对应的pending请求
func (c *Client) deliverResponse(resp *types.Response) {
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

// writeLoop 写入循环
// 在独立goroutine中运行，从发送队列读取数据并发送
func (c *Client) writeLoop() {
	for data := range c.sendCh {
		// 更新队列大小指标
		c.metrics.SendQueueSize.Store(int64(len(c.sendCh)))

		// 设置写入超时
		if c.cfg.WriteTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
		}

		// 发送消息
		if err := c.conn.WriteMessage(c.codec.MessageType(), data); err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			c.metrics.FailedSends.Add(1)
			continue
		}
	}
}

// flushBatch 批量flush回调
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
func (c *Client) OnReceive(fn func(*types.Response)) {
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

// BackoffController 获取背压控制器
func (c *Client) BackoffController() *core.AdaptiveBackoff {
	return c.backoff
}

// BackoffMetrics 获取背压指标
func (c *Client) BackoffMetrics() *core.BackoffMetrics {
	return c.backoffMetrics
}

// hashString 计算字符串的哈希值
// 用于分片索引计算
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
