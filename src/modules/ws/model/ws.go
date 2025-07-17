package model

// WSRequest ws消息接收
type WSRequest struct {
	RequestID string `json:"requestId"` // 请求ID
	Type      string `json:"type"`      // 业务类型
	Data      any    `json:"data"`      // 查询结构
}
