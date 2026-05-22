package server

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/types"
)

// Conn WebSocket 连接（服务端侧）
// 所有可变状态均为 unexported，通过方法访问
type Conn struct {
	id       string          // 连接唯一标识符
	server   *Server         // 所属服务端
	conn     *websocket.Conn // 底层连接
	codec    codec.Codec     // 编解码器
	sendCh   chan []byte     // 发送队列
	lastActive atomic.Int64  // 最后活跃时间（Unix 毫秒）
	failCount  atomic.Int32  // 健康检查连续失败次数

	meta   map[string]any // 连接元数据
	metaMu sync.RWMutex   // 元数据读写锁

	done      chan struct{}
	writeDone chan struct{}
	closeOnce sync.Once
}

// ID 获取连接唯一标识
func (c *Conn) ID() string { return c.id }

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
	c.metaMu.Lock()
	c.meta[key] = val
	c.metaMu.Unlock()
}

// GetMeta 获取元数据（线程安全）
func (c *Conn) GetMeta(key string) (any, bool) {
	c.metaMu.RLock()
	v, ok := c.meta[key]
	c.metaMu.RUnlock()
	return v, ok
}

// init 初始化连接，启动 readLoop/writeLoop/healthLoop
func (c *Conn) init() {
	c.done = make(chan struct{})
	c.writeDone = make(chan struct{})
	c.lastActive.Store(time.Now().UnixMilli())

	c.server.conns.add(c)

	if c.server.onConnect != nil {
		c.server.onConnect(c)
	}

	go c.readLoop()
	go c.writeLoop()

	if c.server.cfg.Heartbeat > 0 {
		go c.healthLoop()
	}
}

// Close 关闭连接
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.server.conns.remove(c)
		close(c.done)

		if c.server.onDisconnect != nil {
			c.server.onDisconnect(c)
		}

		<-c.writeDone
		err = c.conn.Close(websocket.StatusNormalClosure, "connection closed")
	})
	return err
}

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

// SendResp 发送响应（底层方法）
func (c *Conn) SendResp(resp *types.Response) error {
	resp.Ts = time.Now().UnixMilli()
	data, err := c.codec.MarshalResponse(resp)
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

// readLoop 读取循环，handler 异步分发不阻塞
func (c *Conn) readLoop() {
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-c.done:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		c.lastActive.Store(time.Now().UnixMilli())

		if c.server.cfg.MaxMessageSize > 0 && len(data) > c.server.cfg.MaxMessageSize {
			_ = c.SendError("", "invalid_request", 413, "message too large")
			continue
		}

		req, err := c.codec.UnmarshalRequest(data)
		if err != nil {
			_ = c.SendError("", "invalid_request", 400, "invalid request")
			continue
		}

		c.server.handlersMu.RLock()
		handler := c.server.handlers[req.Action]
		c.server.handlersMu.RUnlock()

		if handler == nil {
			_ = c.SendError(req.ID, req.Action, 404, "handler not found")
			continue
		}

		c.dispatch(handler, req)
	}
}

// dispatch 异步分发 handler，通过 WorkerPool 执行
func (c *Conn) dispatch(handler Handler, req *types.Request) {
	job := func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				_ = buf[:n]
				_ = c.SendError(req.ID, req.Action, 500, "internal server error")
			}
		}()
		handler(c, req)
	}

	if c.server.workers != nil {
		c.server.workers.submit(job)
		return
	}
	go job()
}

// writeLoop 写入循环，drain 模式确保关闭前排空发送队列
func (c *Conn) writeLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-c.done:
			cancel()
		case <-ctx.Done():
		}
	}()

	defer close(c.writeDone)

	msgType := websocket.MessageType(c.codec.MessageType())

	for {
		select {
		case <-c.done:
			// drain 剩余消息
			drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
			for {
				select {
				case data := <-c.sendCh:
					_ = c.conn.Write(drainCtx, msgType, data)
				default:
					drainCancel()
					return
				}
			}

		case data := <-c.sendCh:
			if err := c.conn.Write(ctx, msgType, data); err != nil {
				return
			}
		}
	}
}

// healthLoop 健康检查循环
// 每 Heartbeat/2 检查一次，连续 3 次失败后关闭连接
func (c *Conn) healthLoop() {
	pingInterval := c.server.cfg.Heartbeat / 2
	if pingInterval < time.Second {
		pingInterval = time.Second
	}
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			lastActive := time.UnixMilli(c.lastActive.Load())

			// 空闲超时检测
			if time.Since(lastActive) > c.server.cfg.Heartbeat {
				c.failCount.Add(1)
				if c.failCount.Load() >= 3 {
					c.Close()
					return
				}
				continue
			}

			// Ping 探测
			pingCtx, cancel := context.WithTimeout(context.Background(), pingInterval)
			err := c.conn.Ping(pingCtx)
			cancel()

			if err != nil {
				c.failCount.Add(1)
				if c.failCount.Load() >= 3 {
					c.Close()
					return
				}
				continue
			}

			// 成功，重置失败计数
			c.failCount.Store(0)
			c.lastActive.Store(time.Now().UnixMilli())
		}
	}
}
