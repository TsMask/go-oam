package server

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"net/http"

	"github.com/coder/websocket"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/types"
)

// connMetaKey 用于 context 存储 metadata 的内部 key
type connMetaKey struct{}

// Conn WebSocket 连接（服务端侧）
type Conn struct {
	id     string          // 连接唯一标识符
	server *Server         // 所属服务端
	conn   *websocket.Conn // 底层连接
	sendCh chan []byte     // 发送队列

	lastActive atomic.Int64 // 最后活跃时间（Unix 毫秒）

	// codec 配置的编解码器（binary 解码用）
	codec codec.Codec

	// respCodec 自动检测的响应编码器（text→JSON，binary→配置的编码器）
	respMu    sync.RWMutex
	respCodec codec.Codec

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once

	subsMu sync.RWMutex    // 保护 subs
	subs   map[string]bool // 已订阅的 topic 集合
}

// ID 获取连接唯一标识
func (c *Conn) ID() string { return c.id }

// Context 获取连接上下文（取消时连接关闭）
func (c *Conn) Context() context.Context { return c.ctx }

// LastActiveTime 获取最后活跃时间
func (c *Conn) LastActiveTime() time.Time {
	ts := c.lastActive.Load()
	if ts == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ts)
}

// SetMeta 设置元数据（线程安全）
func (c *Conn) SetMeta(key string, val any) {
	c.ctx.Value(connMetaKey{}).(*sync.Map).Store(key, val)
}

// GetMeta 获取元数据（线程安全）
func (c *Conn) GetMeta(key string) (any, bool) {
	return c.ctx.Value(connMetaKey{}).(*sync.Map).Load(key)
}

// CodecName 获取当前连接使用的编码器名称
func (c *Conn) CodecName() string {
	c.respMu.RLock()
	defer c.respMu.RUnlock()
	return c.respCodec.Name()
}

// getRespCodec 获取响应编码器
func (c *Conn) getRespCodec() codec.Codec {
	c.respMu.RLock()
	defer c.respMu.RUnlock()
	return c.respCodec
}

// setRespCodec 设置响应编码器
func (c *Conn) setRespCodec(cc codec.Codec) {
	c.respMu.Lock()
	c.respCodec = cc
	c.respMu.Unlock()
}

// ============================================================================
// 订阅
// ============================================================================

// Subscribe 订阅 topic（幂等，重复订阅不报错）
func (c *Conn) Subscribe(topics ...string) {
	c.subsMu.Lock()
	for _, t := range topics {
		if !c.subs[t] {
			c.subs[t] = true
			c.server.topics.subscribe(t, c)
		}
	}
	c.subsMu.Unlock()
}

// Unsubscribe 取消订阅 topic
func (c *Conn) Unsubscribe(topics ...string) {
	c.subsMu.Lock()
	for _, t := range topics {
		if c.subs[t] {
			delete(c.subs, t)
			c.server.topics.unsubscribe(t, c)
		}
	}
	c.subsMu.Unlock()
}

// Subscriptions 获取当前连接已订阅的所有 topic
func (c *Conn) Subscriptions() []string {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	result := make([]string, 0, len(c.subs))
	for t := range c.subs {
		result = append(result, t)
	}
	return result
}

// ============================================================================
// 生命周期
// ============================================================================

// init 启动连接协程
func (c *Conn) init(r *http.Request) {
	meta := &sync.Map{}
	base := context.WithValue(context.Background(), connMetaKey{}, meta)
	c.ctx, c.cancel = context.WithCancel(base)
	c.lastActive.Store(time.Now().UnixMilli())
	c.subs = make(map[string]bool)

	// 默认用配置的编码器响应
	c.respCodec = c.codec

	c.server.conns.add(c)

	if c.server.onConnect != nil {
		c.server.onConnect(c, r)
	}

	go c.readLoop()
	go c.writeLoop()

	if c.server.cfg.heartbeat > 0 {
		go c.healthLoop()
	}
}

// Close 关闭连接
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.server.conns.remove(c)
		c.server.topics.unsubscribeAll(c)
		c.cancel()

		if c.server.onDisconnect != nil {
			c.server.onDisconnect(c)
		}

		err = c.conn.CloseNow()
	})
	return err
}

// ============================================================================
// 发送
// ============================================================================

// Send 发送响应
func (c *Conn) Send(id, action string, code int32, data []byte) error {
	return c.SendResp(&types.Response{ID: id, Action: action, Code: code, Data: data})
}

// SendOK 发送成功响应（状态码 200）
func (c *Conn) SendOK(id, action string, data []byte) error {
	return c.SendResp(&types.Response{ID: id, Action: action, Code: 200, Data: data})
}

// SendError 发送错误响应
func (c *Conn) SendError(id, action string, code int32, msg string) error {
	return c.SendResp(&types.Response{ID: id, Action: action, Code: code, Msg: msg})
}

// SendResp 发送响应（非阻塞，缓冲区满返回 ErrSendFull）
func (c *Conn) SendResp(resp *types.Response) error {
	resp.Ts = time.Now().UnixMilli()
	cc := c.getRespCodec()
	data, err := cc.MarshalResponse(resp)
	if err != nil {
		return err
	}
	select {
	case c.sendCh <- data:
		return nil
	default:
		return ErrSendFull
	}
}

// ============================================================================
// 内部协程
// ============================================================================

// readLoop 读循环，每条消息起独立协程处理，永不阻塞
// 自动检测消息类型：text → JSON，binary → 配置的编码器
func (c *Conn) readLoop() {
	defer c.Close()

	// 预创建 JSON 编码器，用于解码 text 消息
	jsonCodec := codec.JSON()

	for {
		msgType, data, err := c.conn.Read(c.ctx)
		if err != nil {
			return
		}

		c.lastActive.Store(time.Now().UnixMilli())

		if c.server.cfg.maxMessageSize > 0 && len(data) > c.server.cfg.maxMessageSize {
			_ = c.SendError("", "invalid_request", 413, "message too large")
			continue
		}

		// 根据消息类型自动选择解码器
		var reqCodec codec.Codec
		switch msgType {
		case websocket.MessageText:
			reqCodec = jsonCodec
		default:
			reqCodec = c.codec
		}

		req, err := reqCodec.UnmarshalRequest(data)
		if err != nil {
			_ = c.SendError("", "invalid_request", 400, fmt.Sprintf("message decoded as %s", reqCodec.Name()))
			continue
		}

		// 更新响应编码器，后续发送用对应编码回复
		c.setRespCodec(reqCodec)

		c.server.handlersMu.RLock()
		handler := c.server.handlers[req.Action]
		c.server.handlersMu.RUnlock()

		if handler == nil {
			_ = c.SendError(req.ID, req.Action, 404, "handler not found")
			continue
		}

		// 每条消息独立协程，readLoop 立即回到读取
		go func(h Handler, r *types.Request) {
			defer func() {
				if v := recover(); v != nil {
					_ = c.SendError(r.ID, r.Action, 500, "internal server error")
				}
			}()
			h(c, r)
		}(handler, req)
	}
}

// writeLoop 写循环，根据响应编码器决定消息类型
func (c *Conn) writeLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case data := <-c.sendCh:
			cc := c.getRespCodec()
			msgType := websocket.MessageType(cc.MessageType())
			if err := c.conn.Write(c.ctx, msgType, data); err != nil {
				return
			}
		}
	}
}

// healthLoop 健康检查，定期 Ping 保持连接活跃
// coder/websocket 的 Ping 可与 Read 并发调用
func (c *Conn) healthLoop() {
	interval := c.server.cfg.heartbeat / 2
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fails := 0
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(context.Background(), interval)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				fails++
				if fails >= 3 {
					c.Close()
					return
				}
				continue
			}
			fails = 0
			c.lastActive.Store(time.Now().UnixMilli())
		}
	}
}
