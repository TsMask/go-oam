package ws

import (
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/core"
)

// ============================================================================
// 类型定义
// ============================================================================

// Handler 消息处理函数类型
// 参数：
//   - conn: WebSocket 连接
//   - req: 请求消息
type Handler func(*Conn, *Request)

// Middleware 中间件函数类型
// 参数：next 下一个 Handler
// 返回：包装后的 Handler
type Middleware func(Handler) Handler

// connShardCount 分片数量
// 用于连接管理器的分片锁，减少锁竞争
const connShardCount = 256

// serverConfig 内部配置（不对外暴露）
// 通过 functional options 模式配置
type serverConfig struct {
	maxConns          int
	sendBufferSize    int
	recvBufferSize    int
	workerPoolSize    int
	jobQueueSize      int
	batchEnabled      bool
	batchSize         int
	batchTimeout      time.Duration
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	rateLimit         float64
	enableMetrics     bool
	allowedOriginFunc func(origin string) bool // 来源验证函数，默认返回 true
	maxMessageSize    int                      // 最大消息大小（字节），0 表示不限制
}

// ServerMetrics 服务端性能指标
// 使用 atomic 类型保证并发安全
type ServerMetrics struct {
	Connections     atomic.Int64 // 当前连接数
	MessagesTotal   atomic.Int64 // 消息总数
	MessageDuration atomic.Int64 // 消息处理时间（微秒）
	WorkerQueueSize atomic.Int64 // Worker 队列大小
	ActiveWorkers   atomic.Int64 // 活跃 Worker 数
	ErrorsTotal     atomic.Int64 // 错误总数
	BytesSent       atomic.Int64 // 发送字节数
	BytesReceived   atomic.Int64 // 接收字节数
}

// connManager 连接管理器
// 使用分片锁优化并发访问性能
type connManager struct {
	shards [connShardCount]struct {
		mu sync.RWMutex
		m  map[*Conn]struct{} // 存储 Conn 指针
	}
	total atomic.Int64
}

// Server WebSocket 服务端
// 核心组件，负责：
//   - 管理连接生命周期
//   - 注册处理器和中间件
//   - 提供 http.Handler 接口
type Server struct {
	codec    codec.Codec        // 编解码器
	upgrader websocket.Upgrader // WebSocket 升级器

	conns *connManager // 连接管理器

	handlers  map[string]Handler  // 消息处理器
	middleware []Middleware      // 中间件列表

	workers   *core.WorkerPool        // Worker 池（用于批量处理）
	rateLimit *core.AtomicRateLimiter // 限流器

	cfg serverConfig // 内部配置

	onConnect    func(*Conn)    // 连接建立回调
	onDisconnect func(*Conn)    // 连接断开回调

	metrics *ServerMetrics // 性能指标

	closed atomic.Bool // 关闭标记
}

// ============================================================================
// 连接管理器方法
// ============================================================================

// Count 获取当前连接数
func (cm *connManager) Count() int64 {
	return cm.total.Load()
}

// shardIdx 计算分片索引
// 使用 ID 的哈希值来选择分片，确保同一连接每次都映射到同一分片
func (cm *connManager) shardIdx(id string) int {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	return h % connShardCount
}

// add 添加连接
func (cm *connManager) add(c *Conn) {
	idx := cm.shardIdx(c.ID)

	cm.shards[idx].mu.Lock()
	cm.shards[idx].m[c] = struct{}{}
	cm.shards[idx].mu.Unlock()

	cm.total.Add(1)

	// 更新指标
	if c.Server != nil && c.Server.metrics != nil {
		c.Server.metrics.Connections.Add(1)
	}
}

// remove 移除连接
func (cm *connManager) remove(c *Conn) {
	idx := cm.shardIdx(c.ID)

	cm.shards[idx].mu.Lock()
	delete(cm.shards[idx].m, c)
	cm.shards[idx].mu.Unlock()

	cm.total.Add(-1)

	// 更新指标
	if c.Server != nil && c.Server.metrics != nil {
		c.Server.metrics.Connections.Add(-1)
	}
}

// ============================================================================
// Server 公共方法
// ============================================================================

// ConnManager 获取连接管理器
func (s *Server) ConnManager() *connManager {
	return s.conns
}

// Metrics 获取性能指标
func (s *Server) Metrics() *ServerMetrics {
	return s.metrics
}

// RateLimiter 获取限流器
func (s *Server) RateLimiter() *core.AtomicRateLimiter {
	return s.rateLimit
}

// ServeHTTP 实现 http.Handler 接口
// 用于接入标准 HTTP 服务器
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查是否已关闭
	if s.closed.Load() {
		http.Error(w, "server closed", http.StatusServiceUnavailable)
		return
	}

	// 检查连接数限制
	if s.cfg.maxConns > 0 && s.conns.Count() >= int64(s.cfg.maxConns) {
		http.Error(w, "max connections reached", http.StatusServiceUnavailable)
		return
	}

	// 检查限流
	if s.rateLimit != nil && !s.rateLimit.Allow() {
		http.Error(w, "rate limited", http.StatusServiceUnavailable)
		return
	}

	// 升级为 WebSocket 连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusInternalServerError)
		return
	}

	// 创建并初始化连接
	c := s.newConn(conn, r.RemoteAddr)
	c.init()
}

// newConn 创建新连接
func (s *Server) newConn(conn *websocket.Conn, remoteAddr string) *Conn {
	id := gonanoid.Must(21)

	c := &Conn{
		ID:     id,
		Server: s,
		conn:   conn,
		codec:  s.codec,
		SendCh: make(chan []byte, s.cfg.sendBufferSize),
		Meta:   make(map[string]any),
	}

	// 设置元数据
	c.Meta["remote_addr"] = remoteAddr
	c.Meta["connected_at"] = time.Now()

	return c
}

// NewServer 创建 WebSocket 服务端
// 参数：
//   - codec: 编解码器
//   - opts: 配置选项
func NewServer(codec codec.Codec, opts ...ServerOption) *Server {
	cfg := serverConfig{
		maxConns:          100000,
		sendBufferSize:    1000,
		recvBufferSize:    4096,
		workerPoolSize:    runtime.NumCPU() * 2,
		jobQueueSize:      10000,
		batchEnabled:      false,
		batchSize:         100,
		batchTimeout:      5 * time.Millisecond,
		heartbeatInterval: 30 * time.Second,
		heartbeatTimeout:  60 * time.Second,
		rateLimit:         100000,
		enableMetrics:     true,
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(&cfg)
	}

	s := &Server{
		codec:    codec,
		cfg:      cfg,
		conns:    new(connManager),
		handlers: make(map[string]Handler),
		metrics:  &ServerMetrics{},
	}

	// 初始化连接管理器
	for i := range s.conns.shards {
		s.conns.shards[i].m = make(map[*Conn]struct{})
	}

	// 初始化 Worker 池
	if cfg.workerPoolSize > 0 {
		s.workers = core.NewWorkerPool(cfg.workerPoolSize, cfg.jobQueueSize)
	}

	// 初始化限流器
	if cfg.rateLimit > 0 {
		s.rateLimit = core.NewAtomicRateLimiter(cfg.rateLimit, cfg.maxConns)
	}

	// 初始化 WebSocket 升级器
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  cfg.recvBufferSize,
		WriteBufferSize: cfg.sendBufferSize,
		CheckOrigin: s.checkOrigin,
	}

	return s
}

// checkOrigin 验证请求来源
func (s *Server) checkOrigin(r *http.Request) bool {
	if s.cfg.allowedOriginFunc == nil {
		return true // 默认允许所有跨域
	}
	return s.cfg.allowedOriginFunc(r.Header.Get("Origin"))
}

// Use 注册中间件
// 中间件按注册顺序执行，类似于 Gin 的中间件
func (s *Server) Use(middleware ...Middleware) {
	s.middleware = append(s.middleware, middleware...)
}

// Handle 注册消息处理器
// 处理器在消息到达时被调用
//
// 参数：
//   - action: 动作类型
//   - handler: 处理函数
func (s *Server) Handle(action string, handler Handler) {
	// 从最后一个中间件开始包装，形成洋葱模型
	var wrapped Handler = handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		wrapped = s.middleware[i](wrapped)
	}
	s.handlers[action] = wrapped
}

// OnConnect 设置连接建立回调
func (s *Server) OnConnect(fn func(*Conn)) {
	s.onConnect = fn
}

// OnDisconnect 设置连接断开回调
func (s *Server) OnDisconnect(fn func(*Conn)) {
	s.onDisconnect = fn
}