package nms

import (
	"github.com/tsmask/go-oam/nms/client"
	serverpkg "github.com/tsmask/go-oam/nms/internal/server"
	"github.com/tsmask/go-oam/nms/internal/registry"
	"github.com/tsmask/go-oam/nms/internal/session"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// NMS 模块入口
// ============================================

// Server NMS 服务端
type Server = serverpkg.Server

// Client NE 客户端
type Client = client.Client

// SessionManager 会话管理器
type SessionManager = session.Manager

// Registry 注册中心
type Registry = registry.Registry

// NE 注册的 NE 信息
type NE = registry.NE

// SessionContext 会话上下文
type SessionContext = session.Context

// ============================================
// 四大功能管理器
// ============================================

type (
	ConfigManager  = serverpkg.ConfigManager
	AlarmManager    = serverpkg.AlarmManager
	MetricsManager  = serverpkg.MetricsManager
	CommandManager  = serverpkg.CommandManager
	AlarmFilter     = serverpkg.AlarmFilter
	ThresholdConfig = serverpkg.ThresholdConfig
)

// ============================================
// 类型别名（简化调用）
// ============================================

type (
	AlarmData         = types.AlarmData
	AlarmResponse     = types.AlarmResponse
	MetricItem        = types.MetricItem
	MetricsResponse   = types.MetricsResponse
	CommandRequest     = types.CommandRequest
	CommandResponse    = types.CommandResponse
	AlarmFilterOptions = types.AlarmFilter
)

// ============================================
// 构造函数
// ============================================

// NewServer 创建 NMS 服务端
func NewServer() *Server {
	return serverpkg.NewServer()
}

// NewClient 创建 NE 客户端
func NewClient(opts ...client.ClientOption) *Client {
	return client.New(opts...)
}

// ============================================
// 客户端选项（导出方便调用）
// ============================================

var (
	WithAddr      = client.WithAddr
	WithNEID      = client.WithNEID
	WithNEType    = client.WithNEType
	WithTLSConfig = client.WithTLSConfig
)

// ============================================
// 辅助函数
// ============================================

// NewRegistry 创建注册中心
func NewRegistry() *Registry {
	return registry.New()
}

// NewSessionManager 创建会话管理器
func NewSessionManager() *SessionManager {
	return session.NewManager()
}

// AlarmSeverity 告警严重程度常量
var AlarmSeverity = types.AlarmSeverity

// Version 版本
const Version = "1.0.0"