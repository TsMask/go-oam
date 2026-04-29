package ws

// State 连接状态
type State int

// 连接状态常量
const (
	StateInit          State = iota // 初始状态
	StateConnecting                 // 连接中
	StateConnected                  // 已连接
	StateDisconnecting              // 断开中
	StateDisconnected               // 已断开
	StateReconnecting               // 重连中
	StateFailed                     // 连接失败
)

// String 返回状态字符串
func (s State) String() string {
	switch s {
	case StateInit:
		return "init"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateReconnecting:
		return "reconnecting"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// IsValid 检查状态是否有效
func (s State) IsValid() bool {
	return s >= StateInit && s <= StateFailed
}

// IsActive 检查是否活跃
func (s State) IsActive() bool {
	return s == StateConnected
}

// IsClosed 检查是否已关闭
func (s State) IsClosed() bool {
	return s == StateDisconnected || s == StateFailed
}

// CanConnect 检查是否可以连接
func (s State) CanConnect() bool {
	return s == StateInit || s == StateDisconnected || s == StateFailed
}
