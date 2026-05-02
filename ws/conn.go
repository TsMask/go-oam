package ws

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/types"
)

// Conn WebSocket 连接
// 封装底层的 websocket.Conn，提供：
//   - 发送队列 (SendCh) 用于异步发送
//   - 元数据管理 (Meta)
//   - 连接生命周期管理 (readLoop/writeLoop/healthLoop)
//   - Graceful shutdown 支持
type Conn struct {
	ID     string          // 连接唯一标识符 (Nanoid)
	Server *Server         // 所属服务端引用
	conn   *websocket.Conn // 底层 WebSocket 连接
	codec  codec.Codec     // 编解码器（复制自 Server）

	// SendCh 发送队列
	// 容量由 serverConfig.sendBufferSize 决定
	// 使用 channel 实现异步发送，writeLoop 从此 channel 消费消息
	SendCh chan []byte

	// LastActive 最后活跃时间
	// 用于心跳检测和连接超时判断
	LastActive time.Time

	// Meta 连接元数据
	// 用于存储连接相关的信息，如远程地址、用户信息等
	Meta map[string]any

	// handlers 消息处理器映射表
	// 由 Server 在创建连接时注入
	handlers map[string]Handler

	// done 关闭信号 channel
	// 用于通知所有循环 graceful shutdown
	done chan struct{}

	// writeDone writeLoop 退出信号
	// writeLoop 在排空 SendCh 并退出后关闭此 channel
	// Close() 等待此信号后再关闭底层连接
	writeDone chan struct{}

	// closed 关闭标记
	closed atomic.Bool

	// health 健康检查状态（暂时设为 nil，后续可以加健康检查）
	health any
}

// init 初始化连接
// 在连接创建后调用，启动以下 goroutine：
//   - readLoop: 读取并处理消息
//   - writeLoop: 从 SendCh 消费消息并发送
//   - healthLoop: 心跳检测（如果配置了 heartbeatInterval）
func (c *Conn) init() {
	c.done = make(chan struct{})
	c.writeDone = make(chan struct{})

	// 设置初始读超时
	if c.Server.cfg.heartbeatTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.Server.cfg.heartbeatTimeout))
	}

	// 注册到连接管理器
	c.Server.conns.add(c)

	// 触发连接回调
	if c.Server.onConnect != nil {
		c.Server.onConnect(c)
	}

	// 启动处理循环
	go c.readLoop()
	go c.writeLoop()
	if c.Server.cfg.heartbeatInterval > 0 {
		go c.healthLoop()
	}
}

// Close 关闭连接，实现 graceful shutdown
// 1. 标记 closed 为 true
// 2. 关闭 done channel 通知所有循环退出
// 3. 关闭底层 websocket 连接
//
// 注意：writeLoop 会在关闭前排空 SendCh 中的剩余消息
func (c *Conn) Close() error {
	// 防止重复关闭
	if c.closed.Swap(true) {
		return nil
	}

	// 从连接管理器移除
	c.Server.conns.remove(c)

	// 关闭 done 信号，通知各循环退出
	close(c.done)

	// 触发断开回调
	if c.Server.onDisconnect != nil {
		c.Server.onDisconnect(c)
	}

	// 等待 writeLoop 排空 SendCh
	<-c.writeDone

	return c.conn.Close()
}

// Send 发送响应
// 参数：
//   - id: 请求 ID
//   - action: 动作类型
//   - code: 状态码
//   - data: 响应数据
func (c *Conn) Send(id, action string, code int32, data []byte) error {
	return c.SendResp(&types.Response{
		ID:     id,
		Action: action,
		Code:   code,
		Data:   data,
	})
}

// SendOK 发送成功响应（状态码 200）
// 参数：
//   - id: 请求 ID
//   - action: 动作类型
//   - data: 响应数据
func (c *Conn) SendOK(id, action string, data []byte) error {
	return c.SendResp(&types.Response{
		ID:     id,
		Action: action,
		Code:   200,
		Data:   data,
	})
}

// SendError 发送错误响应
// 参数：
//   - id: 请求 ID
//   - action: 动作类型
//   - code: 错误状态码
//   - msg: 错误消息
func (c *Conn) SendError(id, action string, code int32, msg string) error {
	return c.SendResp(&types.Response{
		ID:     id,
		Action: action,
		Code:   code,
		Msg:    msg,
	})
}

// SendResp 发送响应（底层方法）
// 将响应消息编码后发送到 SendCh
//
// 返回值：
//   - nil: 发送成功
//   - ErrSendFull: SendCh 满了（背压）
func (c *Conn) SendResp(resp *types.Response) error {
	// 填充时间戳
	resp.Ts = time.Now().UnixMilli()

	// 编码消息
	data, err := c.codec.MarshalResponse(resp)
	if err != nil {
		return err
	}

	// 非阻塞发送到 SendCh
	select {
	case c.SendCh <- data:
		return nil
	default:
		return ErrSendFull
	}
}

// SetMeta 设置元数据
func (c *Conn) SetMeta(key string, val any) {
	c.Meta[key] = val
}

// GetMeta 获取元数据
// 返回：(value, exists)
func (c *Conn) GetMeta(key string) (any, bool) {
	v, ok := c.Meta[key]
	return v, ok
}

// readLoop 读取循环
// 负责：
// 1. 读取 WebSocket 消息
// 2. 解码请求
// 3. 路由到对应的 Handler
// 4. 处理心跳（Pong）
//
// 注意：此函数在连接断开时 defer 调用 Close()
func (c *Conn) readLoop() {
	defer func() {
		// 连接断开时清理
		c.Close()
	}()

	for {
		// 更新读超时
		if c.Server.cfg.heartbeatTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.Server.cfg.heartbeatTimeout))
		}

		// 读取消息
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			// 读取错误，连接已断开
			return
		}

		// 更新最后活跃时间
		c.LastActive = time.Now()

		// 处理心跳
		if msgType == websocket.PingMessage {
			c.conn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		// 检查消息大小
		if c.Server.cfg.maxMessageSize > 0 && len(data) > c.Server.cfg.maxMessageSize {
			_ = c.SendError("", "invalid_request", 413, "message too large")
			continue
		}

		// 解码请求
		req, err := c.codec.UnmarshalRequest(data)
		if err != nil {
			// 无效请求，发送错误响应但不关闭连接
			_ = c.SendError("", "invalid_request", 400, "invalid request")
			continue
		}

		// 更新消息指标
		if c.Server.metrics != nil {
			c.Server.metrics.MessagesTotal.Add(1)
			c.Server.metrics.BytesReceived.Add(int64(len(data)))
		}

		// 查找处理器
		handler := c.Server.handlers[req.Action]
		if handler == nil {
			_ = c.SendError(req.ID, req.Action, 404, "handler not found")
			continue
		}

		// 调用处理器
		handler(c, req)
	}
}

// writeLoop 写入循环
// 负责从 SendCh 消费消息并发送到 WebSocket 连接
//
// Graceful Shutdown 机制：
// 当 done channel 关闭时（Close() 被调用），会先排空 SendCh 中
// 的所有剩余消息后再退出，确保所有待发送消息都被发送出去。
func (c *Conn) writeLoop() {
	for {
		select {
		case <-c.done:
			// Graceful shutdown：排空发送队列，设置关闭超时
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			for data := range c.SendCh {
				if err := c.conn.WriteMessage(c.codec.MessageType(), data); err != nil {
					// 写入失败，通知 Close() 继续
					close(c.writeDone)
					return
				}
				// 更新发送指标
				if c.Server.metrics != nil {
					c.Server.metrics.BytesSent.Add(int64(len(data)))
				}
			}
			// 排空完成，通知 Close() 可以关闭连接
			close(c.writeDone)
			return

		case data := <-c.SendCh:
			// 正常发送
			if c.Server.cfg.heartbeatTimeout > 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.Server.cfg.heartbeatTimeout))
			}

			if err := c.conn.WriteMessage(c.codec.MessageType(), data); err != nil {
				// 写入失败，连接已断开
				return
			}

			// 更新发送指标
			if c.Server.metrics != nil {
				c.Server.metrics.BytesSent.Add(int64(len(data)))
			}
		}
	}
}

// healthLoop 健康检查循环
// 定期检查连接是否超时（最后活跃时间 + heartbeatTimeout）
// 如果超时则关闭连接
func (c *Conn) healthLoop() {
	ticker := time.NewTicker(c.Server.cfg.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			// 连接已关闭
			return
		case <-ticker.C:
			// 检查是否超时
			if c.Server.cfg.heartbeatTimeout > 0 &&
				time.Since(c.LastActive) > c.Server.cfg.heartbeatTimeout {
				// 超时，关闭连接
				c.Close()
				return
			}
			// 发送心跳
			c.conn.SetWriteDeadline(time.Now().Add(c.Server.cfg.heartbeatTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.Close()
				return
			}
		}
	}
}
