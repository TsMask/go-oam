package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// ClientConn 客户端连接
type ClientConn struct {
	Url           string          // 连接地址 ws:127.0.0.1:8080/ws
	RawQuery      string          // 原始查询参数 encoded query values, without '?'
	LastHeartbeat int64           // 最近一次心跳消息（毫秒）
	SendChan      chan []byte     // 消息通道
	StopChan      chan struct{}   // 停止信号-退出协程
	wsConn        *websocket.Conn // 连接实例
}

// Connect 连接
func (c *ClientConn) Connect() error {
	urlStr := fmt.Sprintf("%s?%s", c.Url, c.RawQuery)
	wsConn, resp, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		return fmt.Errorf("ws client Connect %s %s err: %s", resp.Proto, resp.Status, err.Error())
	}
	c.wsConn = wsConn
	c.LastHeartbeat = time.Now().UnixMilli()
	if c.SendChan == nil {
		c.SendChan = make(chan []byte, 100) // 消息通道现在数量
	}
	if c.StopChan == nil {
		c.StopChan = make(chan struct{}, 1) //  卡死循环标记
	}
	return nil
}

// Close 客户端关闭
func (c *ClientConn) Close() error {
	if c.wsConn == nil {
		return fmt.Errorf("plase ws client connect")
	}
	c.SendChan <- []byte("ws:close")
	c.StopChan <- struct{}{}
	return c.wsConn.Close()
}

// Ping 客户端心跳非消息由客户端协商
func (c *ClientConn) Ping() {
	c.SendChan <- []byte("ws:ping")
}

// Send 客户端发送
func (c *ClientConn) Send(msg []byte) {
	c.SendChan <- msg
}

// SendString 客户端发送字符串
func (c *ClientConn) SendString(str string) {
	c.SendChan <- []byte(str)
}

// SendJSON 客户端发送可序列化为json的对象
func (c *ClientConn) SendJSON(v any) {
	msgByte, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.SendChan <- msgByte
}

// ReadListen 客户端读取消息监听
//
// msgType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
//
// receiveFn 接收函数进行消息处理
func (c *ClientConn) ReadListen(msgType int, errorFn func(error), receiveFn func(*ClientConn, []byte)) {
	defer func() {
		if err := recover(); err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws client ReadMessage Panic Error: %v", err))
			}
		}
	}()
	for {
		if receiveFn == nil {
			return
		}
		// 读取消息
		messageType, msg, err := c.wsConn.ReadMessage()
		if err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws client ReadMessage err: %s", err.Error()))
			}
			c.Close()
			return
		}
		// fmt.Println(messageType, string(msg))

		switch messageType {
		case msgType:
			receiveFn(c, msg)
		}
	}
}

// WriteListen 客户端写入消息监听
//
// msgType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
//
// conn.SendChan <- msgByte 写入消息
func (c *ClientConn) WriteListen(msgType int, errorFn func(error)) {
	defer func() {
		if err := recover(); err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws client WriteMessage Panic Error: %v", err))
			}
		}
	}()
	// 消息发送监听
	for msg := range c.SendChan {
		// PING句柄
		if string(msg) == "ws:ping" {
			c.LastHeartbeat = time.Now().UnixMilli()
			c.wsConn.WriteMessage(websocket.PingMessage, []byte{})
			continue
		}
		// 关闭句柄
		if string(msg) == "ws:close" {
			c.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}

		// 发送消息
		err := c.wsConn.WriteMessage(msgType, msg)
		if err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws client WriteMessage err: %s", err.Error()))
			}
			c.Close()
			return
		}
	}
}
