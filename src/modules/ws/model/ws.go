package model

import "github.com/gorilla/websocket"

// WSClient ws客户端
type WSClient struct {
	ID            string          // 客户端连接ID-随机字符串16位
	Conn          *websocket.Conn // 连接实例
	LastHeartbeat int64           // 最近一次心跳消息（毫秒）
	BindUid       string          // 绑定唯一标识ID
	SubGroup      []string        // 订阅组ID
	MsgChan       chan []byte     // 消息通道
	StopChan      chan struct{}   // 停止信号-退出协程
	ChildConn     any             // 子连接实例-携带某种连接会话
}

// WSRequest ws消息接收
type WSRequest struct {
	RequestID string `json:"requestId"` // 请求ID
	Type      string `json:"type"`      // 业务类型
	Data      any    `json:"data"`      // 查询结构
}
