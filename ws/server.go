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

// Handler 消息处理器函数类型
type Handler func(conn *Conn, req *Request)

// Middleware 中间件函数类型
type Middleware func(next Handler) Handler

// connShardCount 分片数量
const connShardCount = 256

// ServerConfig 服务端配置
type ServerConfig struct {
	MaxConns          int           // 最大连接数
	SendBufferSize    int           // 发送缓冲区大小
	RecvBufferSize    int           // 接收缓冲区大小
	WorkerPoolSize    int           // Worker 池大小
	JobQueueSize      int           // 任务队列大小
	BatchEnabled      bool          // 是否启用批量发送
	BatchSize         int           // 批量发送大小
	BatchTimeout      time.Duration // 批量超时
	HeartbeatInterval time.Duration // 心跳间隔
	HeartbeatTimeout  time.Duration // 心跳超时
	RateLimit         float64       // 限流速率
	EnableMetrics     bool          // 是否启用指标收集
}

// ServerMetrics 服务端性能指标
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

// ConnManager 连接管理器
type ConnManager struct {
	shards [connShardCount]struct {
		mu sync.RWMutex
		m  map[string]*Conn
	}
	total atomic.Int64
}

// Server WebSocket 服务端
type Server struct {
	Addr     string             // 监听地址
	Codec    codec.Codec        // 编解码器
	HttpSrv  *http.Server       // HTTP 服务器
	Upgrader websocket.Upgrader // WebSocket 升级器

	conns *ConnManager // 连接管理器

	handlers   map[string]Handler // 消息处理器
	middleware []Middleware       // 中间件列表

	workers   *core.WorkerPool        // Worker 池
	rateLimit *core.AtomicRateLimiter // 限流器

	healthChecker *core.HealthChecker // 健康检查器
	healthMetrics *core.HealthMetrics // 健康指标

	cfg ServerConfig // 配置

	OnConnect     func(*Conn)       // 连接回调
	OnDisconnect  func(*Conn)       // 断开回调
	OnHealthCheck func(*Conn, bool) // 健康检查回调

	closed atomic.Bool // 关闭标记

	metrics *ServerMetrics // 性能指标
}

// Count 获取当前连接数
func (cm *ConnManager) Count() int64 {
	return cm.total.Load()
}

// ConnManager 获取连接管理器
func (s *Server) ConnManager() *ConnManager {
	return s.conns
}

// Metrics 获取性能指标
func (s *Server) Metrics() *ServerMetrics {
	return s.metrics
}

// HealthMetrics 获取健康指标
func (s *Server) HealthMetrics() *core.HealthMetrics {
	return s.healthMetrics
}

// ServeHTTP 实现 http.Handler 接口
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusInternalServerError)
		return
	}

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
		Codec:  s.Codec,
		SendCh: make(chan []byte, s.cfg.SendBufferSize),
		Meta:   make(map[string]any),
	}

	c.Meta["remote_addr"] = remoteAddr
	c.Meta["connected_at"] = time.Now()

	return c
}

// NewServer 创建 WebSocket 服务端
func NewServer(addr string, codec codec.Codec, opts ...ServerOption) *Server {
	cfg := ServerConfig{
		MaxConns:          100000,
		SendBufferSize:    1000,
		RecvBufferSize:    4096,
		WorkerPoolSize:    runtime.NumCPU() * 2,
		JobQueueSize:      10000,
		BatchEnabled:      false,
		BatchSize:         100,
		BatchTimeout:      5 * time.Millisecond,
		HeartbeatInterval: 30 * time.Second,
		HeartbeatTimeout:  60 * time.Second,
		RateLimit:         100000,
		EnableMetrics:     true,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	s := &Server{
		Addr:     addr,
		Codec:    codec,
		cfg:      cfg,
		conns:    new(ConnManager),
		handlers: make(map[string]Handler),
		metrics:  &ServerMetrics{},
	}

	for i := range s.conns.shards {
		s.conns.shards[i].m = make(map[string]*Conn)
	}

	if cfg.WorkerPoolSize > 0 {
		s.workers = core.NewWorkerPool(cfg.WorkerPoolSize, cfg.JobQueueSize)
	}

	if cfg.RateLimit > 0 {
		s.rateLimit = core.NewAtomicRateLimiter(cfg.RateLimit, cfg.MaxConns)
	}

	s.Upgrader = websocket.Upgrader{
		ReadBufferSize:  cfg.RecvBufferSize,
		WriteBufferSize: cfg.SendBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	s.HttpSrv = &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return s
}

// Start 启动服务端
func (s *Server) Start() error {
	return s.HttpSrv.ListenAndServe()
}

// Stop 停止服务端
func (s *Server) Stop() error {
	s.closed.Store(true)
	return s.HttpSrv.Close()
}

// Use 注册中间件
func (s *Server) Use(middleware ...Middleware) {
	s.middleware = append(s.middleware, middleware...)
}

// Handle 注册消息处理器
func (s *Server) Handle(action string, handler Handler) {
	var wrapped Handler = handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		wrapped = s.middleware[i](wrapped)
	}
	s.handlers[action] = wrapped
}
