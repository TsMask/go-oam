package server

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/tsmask/go-oam/nms/proto"
	"github.com/tsmask/go-oam/nms/internal/registry"
	"github.com/tsmask/go-oam/nms/internal/session"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// Server gRPC 服务端
// ============================================

type Server struct {
	pb.UnimplementedNMSServiceServer

	registry   *registry.Registry
	sessionMgr *session.Manager

	// 四大功能管理器
	configMgr  *ConfigManager
	alarmMgr   *AlarmManager
	metricsMgr *MetricsManager
	commandMgr *CommandManager

	// 回调函数（供外部设置）
	AlarmHandler   func(neID string, alarm *types.AlarmData) error
	MetricsHandler func(neID string, metricsType string, items []*types.MetricItem) error
}

// NewServer 创建 NMS 服务端
func NewServer() *Server {
	srv := &Server{
		registry:   registry.New(),
		sessionMgr: session.NewManager(),
	}

	// 初始化四大功能管理器
	srv.configMgr = NewConfigManager(srv)
	srv.alarmMgr = NewAlarmManager(srv)
	srv.metricsMgr = NewMetricsManager(srv)
	srv.commandMgr = NewCommandManager(srv)

	// 设置默认告警/性能处理回调
	srv.alarmMgr.AddFilter(AlarmFilter{}) // 空过滤器，接受所有
	srv.AlarmHandler = srv.alarmMgr.HandleAlarm
	srv.MetricsHandler = srv.metricsMgr.HandleMetrics

	return srv
}

// ============================================
// 连接管理
// ============================================

func (s *Server) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	// 检查 NE 是否已连接
	if existing := s.sessionMgr.Get(req.NeId); existing != nil {
		// NE 已存在，更新信息
		s.sessionMgr.Unregister(req.NeId)
		s.registry.Unregister(req.NeId)
	}

	// 创建会话
	sessCtx := s.createSession(req.NeId, req.NeType, req.Ip, req.Port, req.Attrs)

	// 注册会话
	s.sessionMgr.Register(sessCtx)

	// 注册到注册中心
	s.registry.Register(sessCtx)

	return &pb.ConnectResponse{
		Code:       0,
		Message:    "success",
		SessionId:  sessCtx.SessionID,
		ServerTime: time.Now().UnixMilli(),
	}, nil
}

func (s *Server) Disconnect(ctx context.Context, req *pb.DisconnectRequest) (*pb.DisconnectResponse, error) {
	// 注销会话
	s.sessionMgr.Unregister(req.NeId)
	s.registry.Unregister(req.NeId)

	return &pb.DisconnectResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// Heartbeat 双向流心跳
func (s *Server) Heartbeat(stream pb.NMSService_HeartbeatServer) error {
	// 等待第一个请求（包含 ne_id 和 session_id）
	firstReq, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive first heartbeat: %v", err)
	}

	neID := firstReq.NeId
	sessCtx := s.sessionMgr.Get(neID)
	if sessCtx == nil {
		return status.Errorf(codes.NotFound, "session not found: %s", neID)
	}

	sessCtx.Status = session.SessionStatusActive

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 更新心跳
		s.sessionMgr.UpdateHeartbeat(neID, req.Status)

		// 发送响应
		if err := stream.Send(&pb.HeartbeatResponse{
			Timestamp: time.Now().UnixMilli(),
			Ack:       true,
			Message:   "ok",
		}); err != nil {
			return err
		}
	}
}

// ============================================
// 配置管理
// ============================================

func (s *Server) PushConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	return s.configMgr.PushConfig(ctx, req)
}

func (s *Server) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.ConfigResponse, error) {
	return s.configMgr.GetConfig(ctx, req)
}

func (s *Server) SyncConfig(ctx context.Context, req *pb.SyncConfigRequest) (*pb.SyncConfigResponse, error) {
	return s.configMgr.SyncConfig(ctx, req)
}

// ============================================
// 性能管理（流式）
// ============================================

func (s *Server) ReportMetrics(stream pb.NMSService_ReportMetricsServer) error {
	// 获取第一个请求（包含 ne_id）
	firstReq, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive: %v", err)
	}

	neID := firstReq.NeId
	sessCtx := s.sessionMgr.Get(neID)
	if sessCtx == nil {
		return status.Errorf(codes.NotFound, "session not found: %s", neID)
	}

	// 处理流式数据
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 客户端关闭流
			return nil
		}
		if err != nil {
			return err
		}

		// 处理性能数据
		if len(req.Metrics) > 0 {
			metrics := convertMetricItems(req.Metrics)
			if s.MetricsHandler != nil {
				if err := s.MetricsHandler(neID, req.MetricsType, metrics); err != nil {
					// 记录错误但不中断流
					continue
				}
			}
		}

		// 发送确认
		if err := stream.Send(&pb.MetricsResponse{
			NeId:    neID,
			Ack:     true,
			Message: "ok",
		}); err != nil {
			return err
		}
	}
}

// ============================================
// 告警管理（流式）
// ============================================

func (s *Server) ReportAlarms(stream pb.NMSService_ReportAlarmsServer) error {
	// 获取第一个请求
	firstReq, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive: %v", err)
	}

	neID := firstReq.NeId
	sessCtx := s.sessionMgr.Get(neID)
	if sessCtx == nil {
		return status.Errorf(codes.NotFound, "session not found: %s", neID)
	}

	// 处理流式告警
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 处理告警
		if req.Alarm != nil {
			alarm := convertAlarm(req.Alarm)
			if s.AlarmHandler != nil {
				if err := s.AlarmHandler(neID, alarm); err != nil {
					continue
				}
			}
		}

		// 发送确认
		if err := stream.Send(&pb.AlarmResponse{
			NeId:    neID,
			AlarmId: req.Alarm.AlarmId,
			Ack:     true,
			Message: "ok",
		}); err != nil {
			return err
		}
	}
}

// ============================================
// 命令控制
// ============================================

func (s *Server) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	return s.commandMgr.ExecuteCommand(ctx, req)
}

// BatchExecuteCommand 服务端流式批量命令
func (s *Server) BatchExecuteCommand(req *pb.BatchCommandRequest, stream pb.NMSService_BatchExecuteCommandServer) error {
	return s.commandMgr.BatchExecuteCommand(req, stream)
}

// ============================================
// 辅助函数
// ============================================

func nowMs() int64 {
	return time.Now().UnixMilli()
}

func convertMetricItems(pbMetrics []*pb.MetricItem) []*types.MetricItem {
	result := make([]*types.MetricItem, len(pbMetrics))
	for i, m := range pbMetrics {
		result[i] = &types.MetricItem{
			Name:   m.Name,
			Value:  m.Value,
			Unit:   m.Unit,
			Labels: m.Labels,
		}
	}
	return result
}

func convertAlarm(pbAlarm *pb.Alarm) *types.AlarmData {
	return &types.AlarmData{
		AlarmID:     pbAlarm.AlarmId,
		AlarmType:   pbAlarm.AlarmType,
		Severity:    pbAlarm.Severity,
		Name:        pbAlarm.Name,
		Description: pbAlarm.Description,
		Source:      pbAlarm.Source,
		StartTime:   pbAlarm.StartTime,
		EndTime:     pbAlarm.EndTime,
		Params:      pbAlarm.Params,
	}
}

// ============================================
// 会话管理（内部使用）
// ============================================

func (s *Server) createSession(neID, neType, ip string, port int32, attrs map[string]string) *session.Context {
	return &session.Context{
		ID:        fmt.Sprintf("sess-%d", nowMs()),
		NEID:      neID,
		SessionID: fmt.Sprintf("sess-%d", nowMs()),
		NEType:    neType,
		IP:        ip,
		Port:      port,
		Attrs:     attrs,
		Status:    session.SessionStatusInit,
	}
}

// ============================================
// 访问器
// ============================================

func (s *Server) Registry() *registry.Registry {
	return s.registry
}

func (s *Server) SessionManager() *session.Manager {
	return s.sessionMgr
}

// ConfigManager 获取配置管理器
func (s *Server) ConfigManager() *ConfigManager {
	return s.configMgr
}

// AlarmManager 获取告警管理器
func (s *Server) AlarmManager() *AlarmManager {
	return s.alarmMgr
}

// MetricsManager 获取性能管理器
func (s *Server) MetricsManager() *MetricsManager {
	return s.metricsMgr
}

// CommandManager 获取命令管理器
func (s *Server) CommandManager() *CommandManager {
	return s.commandMgr
}