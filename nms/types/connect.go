package types

// ============================================
// 连接相关类型
// ============================================

// ConnectRequest 连接请求
type ConnectRequest struct {
	NEID        string            `json:"ne_id"`
	NEType      string            `json:"ne_type"`
	IP          string            `json:"ip"`
	Port        int32             `json:"port"`
	Attrs       map[string]string `json:"attrs"`
	Capabilities map[string]string `json:"capabilities"`
}

// ConnectResponse 连接响应
type ConnectResponse struct {
	Code       int32  `json:"code"`
	Message    string `json:"message"`
	SessionID  string `json:"session_id"`
	ServerTime int64  `json:"server_time"` // 毫秒
}

// DisconnectRequest 断开连接请求
type DisconnectRequest struct {
	NEID      string `json:"ne_id"`
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

// DisconnectResponse 断开连接响应
type DisconnectResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	NEID      string            `json:"ne_id"`
	SessionID string            `json:"session_id"`
	Timestamp int64             `json:"timestamp"` // 毫秒
	Status    map[string]string `json:"status"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Timestamp int64  `json:"timestamp"` // 毫秒
	Ack       bool   `json:"ack"`
	Message   string `json:"message"`
}

// ============================================
// NE 信息
// ============================================

// NEInfo NE 设备信息
type NEInfo struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	IP          string            `json:"ip"`
	Port        int32             `json:"port"`
	Attrs       map[string]string `json:"attrs"`
	Capabilities map[string]string `json:"capabilities"`
	ConnectedAt int64             `json:"connected_at"` // 毫秒
	SessionID   string            `json:"session_id"`
	Online      bool              `json:"online"`
}