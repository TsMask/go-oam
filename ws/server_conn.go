package ws

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/core"
	"github.com/tsmask/go-oam/ws/types"
)

// Conn WebSocket 连接
type Conn struct {
	ID         string          // 连接ID
	Server     *Server         // 所属服务端
	conn       *websocket.Conn // WebSocket 连接
	Codec      codec.Codec     // 编解码器
	SendCh     chan []byte     // 发送队列
	LastActive time.Time       // 最后活跃时间
	Meta       map[string]any  // 元数据

	handlers map[string]Handler  // 处理器映射表
	health   *core.HealthChecker // 健康检查器
}

// init 初始化连接
func (c *Conn) init() {
	if c.Server.cfg.HeartbeatTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.Server.cfg.HeartbeatTimeout))
	}

	go c.readLoop()
	go c.writeLoop()
}

// Close 关闭连接
func (c *Conn) Close() error {
	return c.conn.Close()
}

// Send 发送响应
func (c *Conn) Send(id string, action string, code int32, data []byte) error {
	return c.SendResp(&types.Response{
		ID:     id,
		Action: action,
		Code:   code,
		Data:   data,
	})
}

// SendOK 发送成功响应
func (c *Conn) SendOK(id string, action string, data []byte) error {
	return c.SendResp(&types.Response{
		ID:     id,
		Action: action,
		Code:   200,
		Data:   data,
	})
}

// SendError 发送错误响应
func (c *Conn) SendError(id string, action string, code int32, msg string) error {
	return c.SendResp(&types.Response{
		ID:     id,
		Action: action,
		Code:   code,
		Msg:    msg,
	})
}

// SendResp 发送响应
func (c *Conn) SendResp(resp *types.Response) error {
	data, err := c.Server.Codec.MarshalResponse(resp)
	if err != nil {
		return err
	}
	resp.Ts = time.Now().UnixMilli()
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
func (c *Conn) GetMeta(key string) (any, bool) {
	v, ok := c.Meta[key]
	return v, ok
}

// readLoop 读取循环
func (c *Conn) readLoop() {
	defer c.Close()

	for {
		if c.Server.cfg.HeartbeatTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.Server.cfg.HeartbeatTimeout))
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if c.Server.OnDisconnect != nil {
				c.Server.OnDisconnect(c)
			}
			return
		}

		if msgType == websocket.PingMessage {
			c.conn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		req, err := c.Server.Codec.UnmarshalRequest(data)
		if err != nil {
			_ = c.SendError("", "invalid_request", 400, "invalid request")
			continue
		}

		handler := c.Server.handlers[req.Action]
		if handler == nil {
			_ = c.SendError(req.ID, req.Action, 404, "handler not found")
			continue
		}

		handler(c, req)
	}
}

// writeLoop 写入循环
func (c *Conn) writeLoop() {
	for data := range c.SendCh {
		if c.Server.cfg.HeartbeatTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(c.Server.cfg.HeartbeatTimeout))
		}

		if err := c.conn.WriteMessage(c.Server.Codec.MessageType(), data); err != nil {
			if c.Server.OnDisconnect != nil {
				c.Server.OnDisconnect(c)
			}
			return
		}
	}
}
