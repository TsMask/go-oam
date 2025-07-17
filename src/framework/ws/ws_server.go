package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/src/framework/utils/generate"
)

// ServerConn 服务端连接
type ServerConn struct {
	BindUID       string          // 绑定唯一标识ID
	LastHeartbeat int64           // 最近一次心跳消息（毫秒）
	SendChan      chan []byte     // 消息通道
	StopChan      chan struct{}   // 停止信号-退出协程
	wsConn        *websocket.Conn // 连接实例
	id            string          // 客户端连接ID-随机字符串16位
	anyConn       any             // 子连接实例-携带某种连接会话
}

// Upgrade http升级ws请求
func (c *ServerConn) Upgrade(w http.ResponseWriter, r *http.Request) error {
	wsUpgrader := websocket.Upgrader{
		// 设置消息发送缓冲区大小（byte），如果这个值设置得太小，可能会导致服务端在发送大型消息时遇到问题
		WriteBufferSize: 4096,
		// 消息包启用压缩
		EnableCompression: true,
		// ws握手超时时间
		HandshakeTimeout: 5 * time.Second,
		// ws握手过程中允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	wsConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	c.wsConn = wsConn
	c.LastHeartbeat = time.Now().UnixMilli()
	c.id = generate.Code(16) // 保证在所有服务端中都能保证唯一即可
	if c.SendChan == nil {
		c.SendChan = make(chan []byte, 100)
	}
	if c.StopChan == nil {
		c.StopChan = make(chan struct{}, 1)
	}
	return nil
}

// Close 服务端关闭
func (c *ServerConn) Close() error {
	if c.wsConn == nil {
		return fmt.Errorf("plase upgrade ws conn")
	}
	c.SendChan <- []byte("ws:close")
	c.StopChan <- struct{}{}
	return c.wsConn.Close()
}

// ClientId 客户端连接ID
func (c *ServerConn) ClientId() string {
	return c.id
}

// Pong 客户端心跳非消息由客户端协商
func (c *ServerConn) Pong() {
	c.SendChan <- []byte("ws:pong")
}

// Send 服务端发送
func (c *ServerConn) Send(msg []byte) {
	c.SendChan <- msg
}

// SendString 服务端发送字符串
func (c *ServerConn) SendString(str string) {
	c.SendChan <- []byte(str)
}

// SendJSON 服务端发送可序列化为json的对象
func (c *ServerConn) SendJSON(v any) {
	msgByte, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.SendChan <- msgByte
}

// SetAnyConn 设置子连接实例
func (c *ServerConn) SetAnyConn(anyConn any) {
	c.anyConn = anyConn
}

// AnyConn 获取子连接实例
func (c *ServerConn) GetAnyConn() any {
	return c.anyConn
}

// ReadListen 客户端读取消息监听
//
// msgType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
//
// receiveFn 接收函数进行消息处理
func (c *ServerConn) ReadListen(msgType int, errorFn func(error), receiveFn func(*ServerConn, []byte)) {
	defer func() {
		if err := recover(); err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws ReadMessage UID %s Panic Error: %v", c.BindUID, err))
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
				errorFn(fmt.Errorf("ws ReadMessage UID %s err: %s", c.BindUID, err.Error()))
			}
			c.Close()
			return
		}
		// fmt.Println(messageType, string(msg))

		switch messageType {
		case msgType:
			go receiveFn(c, msg)
		}
	}
}

// WriteListen 客户端写入消息监听
//
// msgType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
//
// conn.SendChan <- msgByte 写入消息
func (c *ServerConn) WriteListen(msgType int, errorFn func(error)) {
	defer func() {
		if err := recover(); err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws WriteMessage UID %s Panic Error: %v", c.BindUID, err))
			}
		}
	}()
	// 消息发送监听
	for msg := range c.SendChan {
		// PONG句柄
		if string(msg) == "ws:pong" {
			c.LastHeartbeat = time.Now().UnixMilli()
			c.wsConn.WriteMessage(websocket.PongMessage, []byte{})
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
				errorFn(fmt.Errorf("ws WriteMessage UID %s err: %s", c.BindUID, err.Error()))
			}
			c.Close()
			return
		}
	}
}
