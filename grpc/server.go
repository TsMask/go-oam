package grpc

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/grpc/protobuf"
	"github.com/tsmask/go-oam/grpc/types"
	"github.com/tsmask/go-oam/ws/core"
)

// connShardCount 分片数量
const connShardCount = 256

// connManager 连接管理器
type connManager struct {
	shards [connShardCount]struct {
		mu sync.RWMutex
		m  map[string]*Conn
	}
	total atomic.Int64
}

func newConnManager() *connManager {
	cm := &connManager{}
	for i := range cm.shards {
		cm.shards[i].m = make(map[string]*Conn)
	}
	return cm
}

func (cm *connManager) shardIdx(id string) int {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	return h % connShardCount
}

func (cm *connManager) add(c *Conn) {
	idx := cm.shardIdx(c.ID)
	cm.shards[idx].mu.Lock()
	cm.shards[idx].m[c.ID] = c
	cm.shards[idx].mu.Unlock()
	cm.total.Add(1)
}

func (cm *connManager) remove(c *Conn) {
	idx := cm.shardIdx(c.ID)
	cm.shards[idx].mu.Lock()
	delete(cm.shards[idx].m, c.ID)
	cm.shards[idx].mu.Unlock()
	cm.total.Add(-1)
}

func (cm *connManager) get(id string) (*Conn, bool) {
	idx := cm.shardIdx(id)
	cm.shards[idx].mu.RLock()
	c, ok := cm.shards[idx].m[id]
	cm.shards[idx].mu.RUnlock()
	return c, ok
}

func (cm *connManager) count() int64 {
	return cm.total.Load()
}

// ServerMetrics 服务端指标
type ServerMetrics struct {
	Connections  atomic.Int64
	MessagesSent atomic.Int64
	MessagesRecv atomic.Int64
	ErrorsTotal  atomic.Int64
}

func newServerMetrics() *ServerMetrics {
	return &ServerMetrics{}
}

// Server gRPC 服务端
type Server struct {
	conns    *connManager
	handlers *serverHandlers

	workers   *core.WorkerPool
	rateLimit *core.AtomicRateLimiter
	metrics   *ServerMetrics

	cfg serverConfig

	onConnect    func(clientID string, meta map[string]string)
	onDisconnect func(clientID string)

	closed atomic.Bool
}

// NewServer 创建 gRPC 服务端
func NewServer(opts ...ServerOption) *Server {
	cfg := serverConfig{
		maxConns:          100000,
		sendBufferSize:    1000,
		workerPoolSize:    runtime.NumCPU() * 2,
		jobQueueSize:      10000,
		heartbeatInterval: 30 * time.Second,
		heartbeatTimeout: 60 * time.Second,
		rateLimit:        100000,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	s := &Server{
		conns:    newConnManager(),
		handlers: newServerHandlers(),
		metrics:  newServerMetrics(),
		cfg:      cfg,
	}

	if cfg.workerPoolSize > 0 {
		s.workers = core.NewWorkerPool(cfg.workerPoolSize, cfg.jobQueueSize)
	}

	if cfg.rateLimit > 0 {
		s.rateLimit = core.NewAtomicRateLimiter(cfg.rateLimit, cfg.maxConns)
	}

	return s
}

// Handle 注册服务端处理函数
func (s *Server) Handle(action string, handler ServerHandler) {
	s.handlers.Handle(action, handler)
}

// OnConnect 设置连接建立回调
func (s *Server) OnConnect(fn func(clientID string, meta map[string]string)) {
	s.onConnect = fn
}

// OnDisconnect 设置连接断开回调
func (s *Server) OnDisconnect(fn func(clientID string)) {
	s.onDisconnect = fn
}

// Metrics 获取服务端指标
func (s *Server) Metrics() *ServerMetrics {
	return s.metrics
}

// ActiveConns 获取当前连接数
func (s *Server) ActiveConns() int64 {
	return s.conns.count()
}

// ClientConn 获取客户端连接
func (s *Server) ClientConn(clientID string) (*Conn, bool) {
	return s.conns.get(clientID)
}

// SendToClient 服务端主动发消息到客户端
func (s *Server) SendToClient(ctx context.Context, clientID, action string, data []byte) ([]byte, error) {
	conn, ok := s.conns.get(clientID)
	if !ok {
		return nil, ErrClientNotFound
	}

	return conn.Call(ctx, action, data)
}

// SendToClientAsync 服务端主动异步发消息到客户端
func (s *Server) SendToClientAsync(ctx context.Context, clientID, action string, data []byte, callback func([]byte, error)) {
	conn, ok := s.conns.get(clientID)
	if !ok {
		callback(nil, ErrClientNotFound)
		return
	}

	conn.CallAsync(ctx, action, data, callback)
}

// ServiceServer gRPC 服务接口实现
type ServiceServer struct {
	protobuf.UnimplementedServiceServer
	server *Server
}

// Stream 实现双向流接口
func (s *ServiceServer) Stream(stream protobuf.Service_StreamServer) error {
	conn := newConn(s.server.cfg.sendBufferSize)

	// 等待首条消息获取 clientID
	pbMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	msg := &types.Message{
		ID:     pbMsg.Id,
		Action: pbMsg.Action,
		Data:   pbMsg.Data,
		Code:   pbMsg.Code,
		Msg:    pbMsg.Msg,
		Ts:     pbMsg.Ts,
	}

	if msg.Action == "register" {
		conn.ID = msg.ID
		if s.server.onConnect != nil {
			s.server.onConnect(conn.ID, nil)
		}
	} else {
		conn.ID = msg.ID
	}

	s.server.conns.add(conn)
	s.server.metrics.Connections.Add(1)

	go s.readLoop(conn, stream)
	go s.writeLoop(conn, stream)

	<-stream.Context().Done()
	s.server.conns.remove(conn)
	s.server.metrics.Connections.Add(-1)

	if s.server.onDisconnect != nil {
		s.server.onDisconnect(conn.ID)
	}

	return nil
}

func (s *ServiceServer) readLoop(conn *Conn, stream protobuf.Service_StreamServer) {
	for {
		pbMsg, err := stream.Recv()
		if err != nil {
			return
		}

		s.server.metrics.MessagesRecv.Add(1)

		msg := &types.Message{
			ID:     pbMsg.Id,
			Action: pbMsg.Action,
			Data:   pbMsg.Data,
			Code:   pbMsg.Code,
			Msg:    pbMsg.Msg,
			Ts:     pbMsg.Ts,
		}

		// 如果是服务端发起的请求（code=0 且 ID 非空），放入 pending 等待响应
		if msg.IsRequest() && msg.ID != "" {
			s.dispatchRequest(conn, msg, stream)
		} else {
			// 否则是客户端发起的调用，路由到 handler
			s.dispatch(conn, msg, stream)
		}
	}
}

func (s *ServiceServer) writeLoop(conn *Conn, stream protobuf.Service_StreamServer) {
	for {
		select {
		case <-conn.done:
			return
		case msg := <-conn.sendCh:
			pbMsg := &protobuf.Message{
				Id:     msg.ID,
				Action: msg.Action,
				Data:   msg.Data,
				Code:   msg.Code,
				Msg:    msg.Msg,
				Ts:     msg.Ts,
			}
			if err := stream.Send(pbMsg); err != nil {
				s.server.metrics.ErrorsTotal.Add(1)
				return
			}
			s.server.metrics.MessagesSent.Add(1)
		}
	}
}

// dispatch 路由客户端请求到 handler
func (s *ServiceServer) dispatch(conn *Conn, msg *types.Message, stream protobuf.Service_StreamServer) {
	if msg.Action == "register" {
		return
	}

	handler, ok := s.server.handlers.Get(msg.Action)
	if !ok {
		resp := &types.Message{}
		resp.SetError(msg.ID, 404, "handler not found")
		conn.sendCh <- resp
		return
	}

	go func() {
		result, err := handler(stream.Context(), conn.ID, msg.Data)
		resp := &types.Message{}
		if err != nil {
			resp.SetError(msg.ID, 500, err.Error())
		} else {
			resp.SetSuccess(msg.ID, result)
		}
		conn.sendCh <- resp
	}()
}

// dispatchRequest 处理服务端发起的请求，等待客户端响应
func (s *ServiceServer) dispatchRequest(conn *Conn, msg *types.Message, stream protobuf.Service_StreamServer) {
	idx := int(hashString(msg.ID) & shardMask)
	respCh := make(chan *types.Message, 1)

	conn.shards[idx].mu.Lock()
	conn.shards[idx].pending[msg.ID] = respCh
	conn.shards[idx].mu.Unlock()

	defer func() {
		conn.shards[idx].mu.Lock()
		delete(conn.shards[idx].pending, msg.ID)
		conn.shards[idx].mu.Unlock()
	}()

	// 将请求发送给客户端
	conn.sendCh <- msg

	// 等待客户端响应
	select {
	case resp := <-respCh:
		// 将客户端响应发送回去
		conn.sendCh <- resp
	case <-stream.Context().Done():
		return
	}
}
