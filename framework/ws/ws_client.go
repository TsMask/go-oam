package ws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/utils/parse"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"google.golang.org/protobuf/proto"
)

// ClientConn 客户端连接
type ClientConn struct {
	id            string          // 客户端连接ID-随机字符串16位
	lastHeartbeat int64           // 最近一次心跳消息（毫秒）
	wsConn        *websocket.Conn // 连接实例
	closeChan     chan struct{}   // 关闭信号-退出协程
	SendChan      chan []byte     // 消息通道 容量默认100
	Url           string          // 连接地址 ws:127.0.0.1:8080/ws
	Heartbeat     time.Duration   // 心跳时间间隔 30秒
}

// Connect 连接
func (c *ClientConn) Connect() error {
	if c.Url == "" {
		return fmt.Errorf("ws client Url is empty")
	}
	wsConn, resp, err := websocket.DefaultDialer.Dial(c.Url, nil)
	if err != nil {
		return fmt.Errorf("ws client Connect %s %s err: %s", resp.Proto, resp.Status, err.Error())
	}
	c.wsConn = wsConn
	c.lastHeartbeat = time.Now().UnixMilli()
	c.closeChan = make(chan struct{}, 1)
	if c.SendChan == nil {
		c.SendChan = make(chan []byte, 100)
	}
	if c.Heartbeat == 0 {
		c.Heartbeat = 30 * time.Second
	}
	return nil
}

// Close 客户端关闭
func (c *ClientConn) Close() error {
	if c.wsConn == nil {
		return fmt.Errorf("plase ws client connect")
	}
	c.SendChan <- []byte("ws:close")
	c.closeChan <- struct{}{}
	return c.wsConn.Close()
}

// CloseSignal 服务端关闭信号
func (c *ClientConn) CloseSignal() <-chan struct{} {
	return c.closeChan
}

// ClientId 客户端连接ID
func (c *ClientConn) ClientId() string {
	return c.id
}

// LastHeartbeat 最近一次心跳消息（毫秒）
func (c *ClientConn) LastHeartbeat() int64 {
	return c.lastHeartbeat
}

// Ping 客户端心跳非消息由客户端协商
func (c *ClientConn) Ping() {
	c.SendChan <- []byte("ws:ping")
}

// SendText 客户端发送文本消息
func (c *ClientConn) SendText(req *protocol.Request) {
	reqByte, err := json.Marshal(req)
	if err != nil {
		return
	}
	c.SendChan <- reqByte
}

// SendTextJSON 客户端发送文本消息为json的对象
func (c *ClientConn) SendTextJSON(uuid string, reqType string, reqData any) {
	var dataByte []byte
	if reqData != nil {
		if v, err := json.Marshal(reqData); err == nil {
			dataByte = v
		}
	}
	c.SendText(&protocol.Request{
		Uuid: uuid,
		Type: reqType,
		Data: dataByte,
	})
}

// SendBinary 客户端发送二进制消息
func (c *ClientConn) SendBinary(req *protocol.Request) {
	reqByte, err := proto.Marshal(req)
	if err != nil {
		return
	}
	c.SendChan <- reqByte
}

// SendBinaryJSON 客户端发送可序列化为json的对象
func (c *ClientConn) SendBinaryJSON(uuid string, reqType string, reqData any) {
	var dataByte []byte
	if reqData != nil {
		if v, err := json.Marshal(reqData); err == nil {
			dataByte = v
		}
	}
	c.SendBinary(&protocol.Request{
		Uuid: uuid,
		Type: reqType,
		Data: dataByte,
	})
}

// SendReqJSON 通过消息类型发送文本消息与二进制消息
//
// messageType 消息类型 websocket.TextMessage=1 websocket.BinaryMessage=2
func (c *ClientConn) SendReqJSON(messageType int, uuid string, reqType string, reqData any) {
	if messageType == websocket.TextMessage {
		c.SendTextJSON(uuid, reqType, reqData)
	} else {
		c.SendBinaryJSON(uuid, reqType, reqData)
	}
}

// ReadListen 客户端读取消息监听
//
// errorFn 接收错误回调函数
// receiveFn 接收消息回调函数
func (c *ClientConn) ReadListen(errorFn func(error), receiveFn func(*ClientConn, int, *protocol.Response)) {
	defer func() {
		c.Close()
		if err := recover(); err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws ReadMessage Panic Error: %v", err))
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
				errorFn(fmt.Errorf("ws ReadMessage Read Error: %v", err))
			}
			c.Close()
			return
		}
		// fmt.Println(messageType, string(msg))

		res := &protocol.Response{}
		switch messageType {
		case websocket.TextMessage:
			if err = json.Unmarshal(msg, res); err != nil {
				if errorFn != nil {
					errorFn(fmt.Errorf("ws ReadMessage json format Error: %v", err))
				}
				continue
			}
		case websocket.BinaryMessage:
			if err = proto.Unmarshal(msg, res); err != nil {
				if errorFn != nil {
					errorFn(fmt.Errorf("ws ReadMessage proto format Error: %v", err))
				}
				continue
			}
		default:
			continue
		}

		// 来自服务端生成的ID
		if res.Code == websocket.PongMessage {
			c.lastHeartbeat = time.Now().UnixMilli()
			c.id = res.Msg
			continue
		}
		go receiveFn(c, messageType, res)
	}
}

// WriteListen 客户端写入消息监听
// conn.SendChan <- msgByte 写入消息
//
// errorFn 接收错误回调函数
func (c *ClientConn) WriteListen(errorFn func(error)) {
	t := time.NewTicker(c.Heartbeat)
	defer func() {
		t.Stop()
		c.Close()
		if err := recover(); err != nil {
			if errorFn != nil {
				errorFn(fmt.Errorf("ws WriteMessage Panic Error: %v", err))
			}
		}
	}()
	for {
		select {
		case <-t.C:
			c.Ping()
		case msg := <-c.SendChan: // 消息发送监听
			// PONG句柄
			if bytes.Equal(msg, []byte("ws:ping")) {
				c.wsConn.WriteMessage(websocket.PingMessage, []byte{})
				continue
			}
			// 关闭句柄
			if bytes.Equal(msg, []byte("ws:close")) {
				closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close")
				c.wsConn.WriteMessage(websocket.CloseMessage, closeMsg)
				return
			}

			// 发送消息
			messageType := websocket.BinaryMessage
			if parse.IsText(msg) {
				messageType = websocket.TextMessage
			}

			if err := c.wsConn.WriteMessage(messageType, msg); err != nil {
				if errorFn != nil {
					errorFn(fmt.Errorf("ws WriteMessage Error: %v", err))
				}
				c.Close()
				return
			}
		}
	}
}
