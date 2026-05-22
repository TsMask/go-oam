package server

import (
	"errors"
	"net/http"
	"runtime"
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

// ErrSendFull 发送通道满（背压）
var ErrSendFull = errors.New("send channel full")

// ============================================================================
// 类型定义
// ============================================================================

// Handler 消息处理函数类型
type Handler func(*Conn, *types.Request)

// Middleware 中间件函数类型
type Middleware func(Handler) Handler

// connShardCount 连接分片数量，降低缓存行抖动
const connShardCount = 64

// ServerConfig 服务端配置
type ServerConfig struct {
	MaxConns          int                      // 最大连接数，默认 100000
	SendBufferSize    int                      // 每连接发送缓冲区大小，默认 1000
	WorkerPoolSize    int                      // Worker 池大小，0 同步执行，默认 NumCPU*2
	JobQueueSize      int                      // 任务队列大小，默认 WorkerPoolSize*10
	Heartbeat         time.Duration            // 心跳间隔，默认 30s，0 禁用
	RateLimit         float64                  // 限流速率（每秒请求数），默认 100000
	AllowedOriginFunc func(origin string) bool // 来源验证函数，默认 nil 允许所有
	MaxMessageSize    int                      // 最大消息大小（字节），0 不限制
}

// ============================================================================
// workerPool — 内联 goroutine 工作池（unexported）
// ============================================================================

type workerPool struct {
	jobCh chan func()
	wg    sync.WaitGroup
}

func newWorkerPool(workers, queueSize int) *workerPool {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	if queueSize <= 0 {
		queueSize = workers * 10
	}
	wp := &workerPool{jobCh: make(chan func(), queueSize)}
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for job := range wp.jobCh {
				if job != nil {
					job()
				}
			}
		}()
	}
	return wp
}

// submit 提交任务，队列满时阻塞等待
func (wp *workerPool) submit(job func()) {
	wp.jobCh <- job
}

// close 关闭工作池并等待所有 worker 退出
func (wp *workerPool) close() {
	close(wp.jobCh)
	wp.wg.Wait()
}

// ============================================================================
// rateLimiter — 内联原子令牌桶限流器（unexported）
// ============================================================================

type rateLimiter struct {
	rate   float64
	burst  int32
	tokens atomic.Int64
	last   atomic.Int64
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	if rate <= 0 {
		rate = 100000
	}
	if burst <= 0 {
		burst = 100000
	}
	rl := &rateLimiter{rate: rate, burst: int32(burst)}
	rl.tokens.Store(int64(burst))
	rl.last.Store(time.Now().UnixNano())
	return rl
}

// allow 检查是否允许请求，CAS 无锁实现
func (rl *rateLimiter) allow() bool {
	for {
		now := time.Now().UnixNano()
		elapsed := float64(now-rl.last.Load()) / float64(time.Second)
		tokens := float64(rl.tokens.Load()) + elapsed*rl.rate
		if tokens > float64(rl.burst) {
			tokens = float64(rl.burst)
		}
		if tokens >= 1 {
			if rl.tokens.CompareAndSwap(int64(tokens), int64(tokens-1)) {
				rl.last.Store(now)
				return true
			}
			continue
		}
		return false
	}
}

// ============================================================================
// ConnManager 连接管理器（分片锁）
// ============================================================================

// ConnManager 连接管理器
type ConnManager struct {
	shards [connShardCount]struct {
		mu sync.RWMutex
		m  map[string]*Conn
	}
	total atomic.Int64
}

// Count 当前连接总数
func (cm *ConnManager) Count() int64 { return cm.total.Load() }

func (cm *ConnManager) shardIdx(id string) int {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	return h & (connShardCount - 1)
}

func (cm *ConnManager) add(c *Conn) {
	idx := cm.shardIdx(c.id)
	cm.shards[idx].mu.Lock()
	cm.shards[idx].m[c.id] = c
	cm.shards[idx].mu.Unlock()
	cm.total.Add(1)
}

func (cm *ConnManager) remove(c *Conn) {
	idx := cm.shardIdx(c.id)
	cm.shards[idx].mu.Lock()
	delete(cm.shards[idx].m, c.id)
	cm.shards[idx].mu.Unlock()
	cm.total.Add(-1)
}

// Get 根据连接 ID 获取连接
func (cm *ConnManager) Get(id string) *Conn {
	idx := cm.shardIdx(id)
	cm.shards[idx].mu.RLock()
	c := cm.shards[idx].m[id]
	cm.shards[idx].mu.RUnlock()
	return c
}

// Range 遍历所有连接，fn 返回 false 停止
func (cm *ConnManager) Range(fn func(*Conn) bool) {
	for i := range cm.shards {
		cm.shards[i].mu.RLock()
		for _, c := range cm.shards[i].m {
			if !fn(c) {
				cm.shards[i].mu.RUnlock()
				return
			}
		}
		cm.shards[i].mu.RUnlock()
	}
}

// ============================================================================
// Server WebSocket 服务端
// ============================================================================

// Server WebSocket 服务端
// 实现 http.Handler 接口，可接入用户自建的 HTTP 服务器
type Server struct {
	codec codec.Codec // 编解码器

	conns *ConnManager

	handlers   map[string]Handler
	handlersMu sync.RWMutex // handlers 读写锁，支持运行时动态注册

	middleware []Middleware

	workers *workerPool  // 内联工作池
	limiter *rateLimiter // 内联限流器

	cfg ServerConfig

	onConnect    func(*Conn)
	onDisconnect func(*Conn)

	closed atomic.Bool
}

// Codec 获取编解码器
func (s *Server) Codec() codec.Codec { return s.codec }

// ConnManager 获取连接管理器
func (s *Server) ConnManager() *ConnManager { return s.conns }

// ============================================================================
// HTTP Handler
// ============================================================================

// ServeHTTP 实现 http.Handler 接口
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.closed.Load() {
		http.Error(w, "server closed", http.StatusServiceUnavailable)
		return
	}

	if s.cfg.MaxConns > 0 && s.conns.Count() >= int64(s.cfg.MaxConns) {
		http.Error(w, "max connections reached", http.StatusServiceUnavailable)
		return
	}

	if s.limiter != nil && !s.limiter.allow() {
		http.Error(w, "rate limited", http.StatusServiceUnavailable)
		return
	}

	if s.cfg.AllowedOriginFunc != nil {
		if !s.cfg.AllowedOriginFunc(r.Header.Get("Origin")) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	c := s.newConn(conn, r.RemoteAddr)
	c.init()
}

// ============================================================================
// 构造与生命周期
// ============================================================================

// NewServer 创建 WebSocket 服务端
func NewServer(codec codec.Codec, opts ...ServerOption) *Server {
	cfg := ServerConfig{
		MaxConns:       100000,
		SendBufferSize: 1000,
		WorkerPoolSize: runtime.NumCPU() * 2,
		JobQueueSize:   0, // 自动计算
		Heartbeat:      30 * time.Second,
		RateLimit:      100000,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	// 自动计算 JobQueueSize
	if cfg.JobQueueSize <= 0 {
		cfg.JobQueueSize = cfg.WorkerPoolSize * 10
	}

	s := &Server{
		codec:    codec,
		cfg:      cfg,
		conns:    new(ConnManager),
		handlers: make(map[string]Handler),
	}

	// 初始化分片 map
	for i := range s.conns.shards {
		s.conns.shards[i].m = make(map[string]*Conn)
	}

	if cfg.WorkerPoolSize > 0 {
		s.workers = newWorkerPool(cfg.WorkerPoolSize, cfg.JobQueueSize)
	}

	if cfg.RateLimit > 0 {
		s.limiter = newRateLimiter(cfg.RateLimit, cfg.MaxConns)
	}

	return s
}

func (s *Server) newConn(raw *websocket.Conn, remoteAddr string) *Conn {
	return &Conn{
		id:       generate.String(21),
		server:   s,
		conn:     raw,
		codec:    s.codec,
		sendCh:   make(chan []byte, s.cfg.SendBufferSize),
		meta:     map[string]any{"remote_addr": remoteAddr, "connected_at": time.Now()},
	}
}

// Shutdown 优雅关闭：停止接受新连接，关闭所有现有连接，等待 Worker 完成
func (s *Server) Shutdown() {
	s.closed.Store(true)

	// 关闭所有现有连接
	s.conns.Range(func(c *Conn) bool {
		c.Close()
		return true
	})

	if s.workers != nil {
		s.workers.close()
	}
}

// ============================================================================
// Handler 注册
// ============================================================================

// Use 注册中间件
func (s *Server) Use(middleware ...Middleware) {
	s.middleware = append(s.middleware, middleware...)
}

// Handle 注册消息处理器（线程安全，支持运行时注册）
func (s *Server) Handle(action string, handler Handler) {
	var wrapped Handler = handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		wrapped = s.middleware[i](wrapped)
	}
	s.handlersMu.Lock()
	s.handlers[action] = wrapped
	s.handlersMu.Unlock()
}

// OnConnect 设置连接建立回调
func (s *Server) OnConnect(fn func(*Conn)) { s.onConnect = fn }

// OnDisconnect 设置连接断开回调
func (s *Server) OnDisconnect(fn func(*Conn)) { s.onDisconnect = fn }

// ============================================================================
// 广播
// ============================================================================

// Broadcast 向所有连接广播消息
func (s *Server) Broadcast(action string, data []byte) {
	resp := &types.Response{Action: action, Code: 200, Data: data}
	s.conns.Range(func(c *Conn) bool {
		_ = c.SendResp(resp)
		return true
	})
}

// BroadcastFilter 向满足条件的连接广播消息
func (s *Server) BroadcastFilter(action string, data []byte, filter func(*Conn) bool) {
	resp := &types.Response{Action: action, Code: 200, Data: data}
	s.conns.Range(func(c *Conn) bool {
		if filter(c) {
			_ = c.SendResp(resp)
		}
		return true
	})
}
