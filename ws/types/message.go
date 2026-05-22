package types

// Request 请求消息
// 客户端发送到服务端的请求结构
//
// 字段说明：
//   - ID: 请求唯一标识符，用于请求-响应匹配
//   - Action: 动作类型，用于路由到不同的处理器
//   - Data: 业务数据，编码格式由 codec 决定
type Request struct {
	ID     string `json:"id"`     // 请求唯一标识符（UUID/Nanoid）
	Action string `json:"action"` // 动作类型，如 "echo", "chat", "subscribe"
	Data   []byte `json:"data"`   // 业务数据
}

// IsValid 检查请求是否有效
// 返回：true 表示请求有效，false 表示无效
// 有效条件：Action 不为空
func (r *Request) IsValid() bool {
	return r.Action != ""
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
	ID     string `json:"id"`             // 请求标识符（原样返回 Request.ID）
	Ts     int64  `json:"ts"`             // 响应时间戳（Unix毫秒）
	Action string `json:"action"`         // 动作类型
	Code   int32  `json:"code"`           // 响应状态码：0成功，200成功，4xx客户端错误，5xx服务端错误
	Msg    string `json:"msg,omitempty"`  // 错误消息
	Data   []byte `json:"data,omitempty"` // 响应数据
}

// IsSuccess 检查响应是否成功
// 返回：true 表示成功，false 表示失败
// 成功条件：Code == 0 或 Code == 200
func (r *Response) IsSuccess() bool {
	return r.Code == 0 || r.Code == 200
}

// Error 获取错误消息
// 返回：如果响应成功返回空字符串，否则返回错误消息
// 用于简化错误处理
func (r *Response) Error() string {
	if r.IsSuccess() {
		return ""
	}
	return r.Msg
}
