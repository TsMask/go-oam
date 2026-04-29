package session

import (
	"sync"
	"time"
)

// ============================================
// Session 会话上下文
// ============================================

type Context struct {
	ID            string            // 会话 ID
	NEID          string            // NE 设备 ID
	SessionID     string            // NMS 分配的 session ID
	NEType        string            // NE 类型
	IP            string            // NE IP
	Port          int32             // NE 端口
	Attrs         map[string]string // 扩展属性
	ConnectedAt   int64             // 毫秒
	LastHeartbeat int64             // 毫秒
	Status        SessionStatus     // 会话状态

	// 流式接口
	AlarmStream   chan<- *AlarmData
	MetricsStream chan<- *MetricsData
}

type SessionStatus int

const (
	SessionStatusInit         SessionStatus = iota
	SessionStatusConnected
	SessionStatusActive
	SessionStatusDisconnecting
	SessionStatusDisconnected
)

func (s SessionStatus) String() string {
	switch s {
	case SessionStatusInit:
		return "init"
	case SessionStatusConnected:
		return "connected"
	case SessionStatusActive:
		return "active"
	case SessionStatusDisconnecting:
		return "disconnecting"
	case SessionStatusDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}

// ============================================
// Manager 会话管理器
// ============================================

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Context // key: ne_id

	// 回调函数
	onConnect    func(*Context)
	onDisconnect func(*Context)
	onHeartbeat  func(*Context, map[string]string)
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Context),
	}
}

// Register 注册会话
func (m *Manager) Register(ctx *Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := nowMs()
	ctx.Status = SessionStatusConnected
	ctx.ConnectedAt = now
	ctx.LastHeartbeat = now
	m.sessions[ctx.NEID] = ctx

	if m.onConnect != nil {
		m.onConnect(ctx)
	}
}

// Unregister 注销会话
func (m *Manager) Unregister(neID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx, ok := m.sessions[neID]; ok {
		ctx.Status = SessionStatusDisconnected
		if m.onDisconnect != nil {
			m.onDisconnect(ctx)
		}
	}
	delete(m.sessions, neID)
}

// Get 获取会话
func (m *Manager) Get(neID string) *Context {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sessions[neID]
}

// List 获取所有会话
func (m *Manager) List() []*Context {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Context, 0, len(m.sessions))
	for _, ctx := range m.sessions {
		result = append(result, ctx)
	}
	return result
}

// UpdateHeartbeat 更新心跳
func (m *Manager) UpdateHeartbeat(neID string, status map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx, ok := m.sessions[neID]; ok {
		ctx.LastHeartbeat = nowMs()
		if m.onHeartbeat != nil {
			m.onHeartbeat(ctx, status)
		}
	}
}

// SetOnConnect 设置连接回调
func (m *Manager) SetOnConnect(fn func(*Context)) {
	m.onConnect = fn
}

// SetOnDisconnect 设置断开回调
func (m *Manager) SetOnDisconnect(fn func(*Context)) {
	m.onDisconnect = fn
}

// SetOnHeartbeat 设置心跳回调
func (m *Manager) SetOnHeartbeat(fn func(*Context, map[string]string)) {
	m.onHeartbeat = fn
}

// Close 关闭会话管理器
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for neID := range m.sessions {
		delete(m.sessions, neID)
	}
}

// ============================================
// 辅助函数
// ============================================

// nowMs 返回当前毫秒时间戳
func nowMs() int64 {
	return int64(time.Now().UnixMilli())
}

// ============================================
// 流式数据类型
// ============================================

type AlarmData struct {
	NEID      string
	SessionID string
	Alarm     *AlarmInfo
	Timestamp int64
}

type MetricsData struct {
	NEID        string
	SessionID   string
	MetricsType string
	Metrics     []*MetricInfo
	Timestamp   int64
}

type AlarmInfo struct {
	AlarmID     string
	AlarmType   string
	Severity    string
	Name        string
	Description string
	Source      string
	StartTime   int64
	EndTime     int64
	Params      map[string]string
}

type MetricInfo struct {
	Name   string
	Value  string
	Unit   string
	Labels map[string]string
}