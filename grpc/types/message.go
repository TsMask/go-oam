package types

import "time"

// Message 统一消息类型
//
// 字段说明：
//   - ID: 请求唯一标识符，用于请求-响应匹配，空ID表示无需响应
//   - Action: 动作类型，用于路由到不同的处理器
//   - Data: 业务数据，编码格式由 codec 决定
//   - Code: 状态码，0=请求(需响应), 200=成功响应, 4xx=客户端错误, 5xx=服务端错误
//   - Msg: 错误消息
//   - Ts: 时间戳
type Message struct {
	ID     string `json:"id"`     // 请求唯一标识符
	Action string `json:"action"` // 动作类型
	Data   []byte `json:"data"`  // 业务数据
	Code   int32  `json:"code"`   // 状态码
	Msg    string `json:"msg"`    // 错误消息
	Ts     int64  `json:"ts"`     // 时间戳
}

// IsRequest 检查是否为请求
// 返回：true 表示请求（需要响应），false 表示响应
// 条件：Code == 0
func (m *Message) IsRequest() bool {
	return m.Code == 0
}

// IsResponse 检查是否为响应
// 返回：true 表示响应，false 表示请求
// 条件：Code == 200
func (m *Message) IsResponse() bool {
	return m.Code == 200
}

// IsSuccess 检查响应是否成功
// 返回：true 表示成功，false 表示失败
// 条件：Code == 0 || Code == 200
func (m *Message) IsSuccess() bool {
	return m.Code == 0 || m.Code == 200
}

// IsError 检查是否为错误
// 返回：true 表示错误
// 条件：Code >= 400
func (m *Message) IsError() bool {
	return m.Code >= 400
}

// Error 获取错误消息
// 返回：如果不是错误返回空字符串
func (m *Message) Error() string {
	if !m.IsError() {
		return ""
	}
	return m.Msg
}

// SetRequest 设置为请求
func (m *Message) SetRequest(id, action string, data []byte) {
	m.ID = id
	m.Action = action
	m.Data = data
	m.Code = 0
	m.Msg = ""
	m.Ts = time.Now().UnixMilli()
}

// SetResponse 设置为响应
func (m *Message) SetResponse(id string, code int32, msg string, data []byte) {
	m.ID = id
	m.Action = ""
	m.Data = data
	m.Code = code
	m.Msg = msg
	m.Ts = time.Now().UnixMilli()
}

// SetSuccess 设置为成功响应
func (m *Message) SetSuccess(id string, data []byte) {
	m.SetResponse(id, 200, "", data)
}

// SetError 设置为错误响应
func (m *Message) SetError(id string, code int32, msg string) {
	m.SetResponse(id, code, msg, nil)
}

// Clone 深拷贝
func (m *Message) Clone() *Message {
	data := make([]byte, len(m.Data))
	copy(data, m.Data)
	return &Message{
		ID:     m.ID,
		Action: m.Action,
		Data:   data,
		Code:   m.Code,
		Msg:    m.Msg,
		Ts:     m.Ts,
	}
}
