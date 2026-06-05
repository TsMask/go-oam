package types

import "encoding/json"

// Request 请求消息
// 客户端发送到服务端的请求结构
//
// 字段说明：
//   - ID: 请求唯一标识符，用于请求-响应匹配
//   - Action: 动作类型，用于路由到不同的处理器
//   - Data: 业务数据，编码格式由 codec 决定
type Request struct {
	ID     string          `json:"id"`     // 请求唯一标识符（UUID/Nanoid）
	Action string          `json:"action"` // 动作类型，如 "echo", "chat", "subscribe"
	Data   json.RawMessage `json:"data"`   // 业务数据（JSON格式）
}

// Response 响应消息
// 服务端返回给客户端的响应结构
//
// 字段说明：
//   - ID: 请求标识符，原样返回 Request.ID，用于匹配请求
//   - Ts: 响应时间戳，Unix时间戳（毫秒）
//   - Action: 动作类型，用于标识响应消息类型（如广播、通知）
//   - Code: 响应状态码，0表示成功
//   - Msg: 错误消息，当 Code != 0 时填充
//   - Data: 响应数据
type Response struct {
	ID     string          `json:"id,omitempty"`     // 请求标识符（原样返回 Request.ID）
	Ts     int64           `json:"ts"`               // 响应时间戳（Unix毫秒）
	Action string          `json:"action,omitempty"` // 动作类型，如 "echo", "chat", "subscribe"
	Code   int32           `json:"code"`             // 响应状态码（200成功，4xx客户端错误，5xx服务端错误）
	Msg    string          `json:"msg,omitempty"`    // 错误消息
	Data   json.RawMessage `json:"data,omitempty"`   // 响应数据（JSON格式）
}
