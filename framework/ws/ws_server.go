package ws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/generate"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"google.golang.org/protobuf/proto"
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
		WriteBufferSize: 4 * 1024,
		ReadBufferSize:  4 * 1024,
		// 子协议字段
		Subprotocols: []string{"oam-ws"},
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
		return fmt.Errorf("plase upgrade conn to websocket conn")
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

// SendText 服务端发送文本消息
func (c *ServerConn) SendText(res *protocol.Response) {
	res.Timestamp = time.Now().UnixMilli()
	resByte, err := json.Marshal(res)
	if err != nil {
		return
	}
	c.SendChan <- resByte
}

// SendTextJSON 服务端发送文本消息为json的对象
func (c *ServerConn) SendTextJSON(uuid string, code int32, msg string, data any) {
	var dataByte []byte
	if data != nil {
		if v, err := json.Marshal(data); err != nil {
			dataByte = v
		}
	}
	c.SendText(&protocol.Response{
		Uuid: uuid,
		Code: code,
		Msg:  msg,
		Data: dataByte,
	})
}

// SendBinary 服务端发送二进制消息
func (c *ServerConn) SendBinary(res *protocol.Response) {
	res.Timestamp = time.Now().UnixMilli()
	resByte, err := proto.Marshal(res)
	if err != nil {
		return
	}
	c.SendChan <- resByte
}

// SendBinaryJSON 服务端发送可序列化为json的对象
func (c *ServerConn) SendBinaryJSON(uuid string, code int32, msg string, data any) {
	var dataByte []byte
	if data != nil {
		if v, err := json.Marshal(data); err != nil {
			dataByte = v
		}
	}
	c.SendBinary(&protocol.Response{
		Uuid: uuid,
		Code: code,
		Msg:  msg,
		Data: dataByte,
	})
}

// SendRespJSON 通过消息类型发送文本消息与二进制消息
//
// messageType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
func (c *ServerConn) SendRespJSON(messageType int, uuid string, code int32, msg string, data any) {
	if messageType == websocket.TextMessage {
		c.SendTextJSON(uuid, code, msg, data)
	} else {
		c.SendBinaryJSON(uuid, code, msg, data)
	}
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
// errorFn 接收错误回调函数
// receiveFn 接收消息回调函数
func (c *ServerConn) ReadListen(errorFn func(error), receiveFn func(*ServerConn, int, *protocol.Request)) {
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

		// 解析请求
		req := &protocol.Request{}
		switch messageType {
		case websocket.TextMessage:
			err = json.Unmarshal(msg, req)
			if err != nil {
				c.SendTextJSON("", resp.CODE_ERROR, "message data format error", nil)
				continue
			}
		case websocket.BinaryMessage:
			err = proto.Unmarshal(msg, req)
			if err != nil {
				c.SendBinaryJSON("", resp.CODE_ERROR, "message data format error", nil)
				continue
			}
		default:
			c.SendChan <- []byte("ws:pong")
			continue
		}

		// 必传uuid确认消息
		if req.Uuid == "" {
			if messageType == websocket.TextMessage {
				c.SendTextJSON("", resp.CODE_ERROR, "message uuid is required", nil)
			} else {
				c.SendBinaryJSON("", resp.CODE_ERROR, "message uuid is required", nil)
			}
			return
		}

		// 默认业务类型
		switch req.Type {
		case "close", "CLOSE":
			c.Close()
			return
		case "ping", "PING":
			c.Pong()
			if messageType == websocket.TextMessage {
				c.SendTextJSON(req.Uuid, resp.CODE_SUCCESS, "PONG", nil)
			} else {
				c.SendBinaryJSON(req.Uuid, resp.CODE_SUCCESS, "PONG", nil)
			}
			continue
		}
		go receiveFn(c, messageType, req)
	}
}

// WriteListen 客户端写入消息监听
// conn.SendChan <- msgByte 写入消息
//
// errorFn 接收错误回调函数
func (c *ServerConn) WriteListen(errorFn func(error)) {
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
		if bytes.Equal(msg, []byte("ws:pong")) {
			c.LastHeartbeat = time.Now().UnixMilli()
			c.wsConn.WriteMessage(websocket.PongMessage, []byte{})
			continue
		}
		// 关闭句柄
		if bytes.Equal(msg, []byte("ws:close")) {
			c.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}

		// 发送消息
		messageType := websocket.BinaryMessage
		if utf8.Valid(msg) {
			messageType = websocket.TextMessage
		}
		err := c.wsConn.WriteMessage(messageType, msg)
		if err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws WriteMessage UID %s err: %s", c.BindUID, err.Error()))
			}
			c.Close()
			return
		}
	}
}
