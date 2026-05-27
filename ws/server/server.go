package server

import (
	"errors"
	"net/http"
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

// ============================================================================
// ConnManager 连接管理器
// ============================================================================

type ConnManager struct {
	mu    sync.RWMutex
	m     map[string]*Conn
	total atomic.Int64
}

func newConnManager() *ConnManager {
	return &ConnManager{m: make(map[string]*Conn)}
}

// Count 当前连接总数
func (cm *ConnManager) Count() int64 { return cm.total.Load() }

// Get 根据连接 ID 获取连接
func (cm *ConnManager) Get(id string) *Conn {
	cm.mu.RLock()
	c := cm.m[id]
	cm.mu.RUnlock()
	return c
}

// Range 遍历所有连接，fn 返回 false 停止
func (cm *ConnManager) Range(fn func(*Conn) bool) {
	cm.mu.RLock()
	conns := make([]*Conn, 0, len(cm.m))
	for _, c := range cm.m {
		conns = append(conns, c)
	}
	cm.mu.RUnlock()

	for _, c := range conns {
		if !fn(c) {
			return
		}
	}
}

func (cm *ConnManager) add(c *Conn) {
	cm.mu.Lock()
	cm.m[c.id] = c
	cm.mu.Unlock()
	cm.total.Add(1)
}

func (cm *ConnManager) remove(c *Conn) {
	cm.mu.Lock()
	delete(cm.m, c.id)
	cm.mu.Unlock()
	cm.total.Add(-1)
}

// ============================================================================
// topicManager 订阅管理器
// ============================================================================

// topicManager 管理 topic → 订阅连接的映射
type topicManager struct {
	mu     sync.RWMutex
	topics map[string]map[string]*Conn // topic → connID → Conn
}

func newTopicManager() *topicManager {
	return &topicManager{topics: make(map[string]map[string]*Conn)}
}

// subscribe 添加订阅
func (tm *topicManager) subscribe(topic string, c *Conn) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	subs, ok := tm.topics[topic]
	if !ok {
		subs = make(map[string]*Conn)
		tm.topics[topic] = subs
	}
	subs[c.id] = c
}

// unsubscribe 取消订阅
func (tm *topicManager) unsubscribe(topic string, c *Conn) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	subs, ok := tm.topics[topic]
	if !ok {
		return
	}
	delete(subs, c.id)
	if len(subs) == 0 {
		delete(tm.topics, topic)
	}
}

// unsubscribeAll 清除某连接的所有订阅
func (tm *topicManager) unsubscribeAll(c *Conn) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for topic, subs := range tm.topics {
		delete(subs, c.id)
		if len(subs) == 0 {
			delete(tm.topics, topic)
		}
	}
}

// subscribers 获取某 topic 的订阅者列表（快照）
func (tm *topicManager) subscribers(topic string) []*Conn {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	subs, ok := tm.topics[topic]
	if !ok {
		return nil
	}
	result := make([]*Conn, 0, len(subs))
	for _, c := range subs {
		result = append(result, c)
	}
	return result
}

// topicCount 获取某 topic 的订阅者数量
func (tm *topicManager) topicCount(topic string) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	subs, ok := tm.topics[topic]
	if !ok {
		return 0
	}
	return len(subs)
}

// topics 获取所有有订阅者的 topic 列表
func (tm *topicManager) topicList() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]string, 0, len(tm.topics))
	for topic := range tm.topics {
		result = append(result, topic)
	}
	return result
}

// ============================================================================
// Server WebSocket 服务端
// ============================================================================

// Server WebSocket 服务端
// 实现 http.Handler 接口，可接入用户自建的 HTTP 服务器
type Server struct {
	codec      codec.Codec        // 编解码器
	conns      *ConnManager       // 连接管理器
	topics     *topicManager      // 订阅管理器
	handlers   map[string]Handler // 消息处理器映射，key 为 Request.Action
	handlersMu sync.RWMutex       // handlers 读写锁，支持运行时动态注册
	middleware []Middleware       // 中间件链，按注册顺序执行

	onConnect    func(*Conn, *http.Request)
	onDisconnect func(*Conn)

	cfg    serverConfig // 配置项
	closed atomic.Bool  // 关闭标志，true 表示已关闭
}

// Codec 获取编解码器
func (s *Server) Codec() codec.Codec { return s.codec }

// ConnManager 获取连接管理器
func (s *Server) ConnManager() *ConnManager { return s.conns }

// ServeHTTP 实现 http.Handler 接口
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.closed.Load() {
		http.Error(w, "server closed", http.StatusServiceUnavailable)
		return
	}

	if s.cfg.maxConns > 0 && s.conns.Count() >= int64(s.cfg.maxConns) {
		http.Error(w, "max connections reached", http.StatusServiceUnavailable)
		return
	}

	if s.cfg.allowedOriginFunc != nil {
		if !s.cfg.allowedOriginFunc(r.Header.Get("Origin")) {
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

	c := &Conn{
		id:     generate.String(21),
		server: s,
		conn:   conn,
		codec:  s.codec,
		sendCh: make(chan []byte, s.cfg.sendBufferSize),
	}
	c.init(r)
	c.SetMeta("remote_addr", r.RemoteAddr)
	c.SetMeta("user_agent", r.UserAgent())
	c.SetMeta("connected_at", time.Now())
}

// NewServer 创建 WebSocket 服务端
func NewServer(opts ...ServerOption) *Server {
	cfg := serverConfig{
		maxConns:       100000,
		sendBufferSize: 1000,
		heartbeat:      30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.codec == "" {
		cfg.codec = "json"
	}

	return &Server{
		codec:    codec.NewCodec(cfg.codec),
		cfg:      cfg,
		conns:    newConnManager(),
		topics:   newTopicManager(),
		handlers: make(map[string]Handler),
	}
}

// Shutdown 优雅关闭
// 标记关闭阻止新连接，遍历关闭所有已有连接
func (s *Server) Shutdown() {
	s.closed.Store(true)
	s.conns.Range(func(c *Conn) bool {
		_ = c.Close()
		return true
	})
}

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
func (s *Server) OnConnect(fn func(*Conn, *http.Request)) { s.onConnect = fn }

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

// ============================================================================
// 订阅/发布
// ============================================================================

// Publish 向某 topic 的所有订阅者发布消息
func (s *Server) Publish(topic string, data []byte) {
	resp := &types.Response{Action: topic, Code: 200, Data: data}
	for _, c := range s.topics.subscribers(topic) {
		_ = c.SendResp(resp)
	}
}

// PublishFilter 向某 topic 中满足条件的订阅者发布消息
func (s *Server) PublishFilter(topic string, data []byte, filter func(*Conn) bool) {
	resp := &types.Response{Action: topic, Code: 200, Data: data}
	for _, c := range s.topics.subscribers(topic) {
		if filter(c) {
			_ = c.SendResp(resp)
		}
	}
}

// TopicCount 获取某 topic 的订阅者数量
func (s *Server) TopicCount(topic string) int {
	return s.topics.topicCount(topic)
}

// Topics 获取所有有订阅者的 topic 列表
func (s *Server) Topics() []string {
	return s.topics.topicList()
}
