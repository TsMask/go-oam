package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	pb "github.com/tsmask/go-oam/nms/proto"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// ConfigManager 配置管理器
// ============================================

type ConfigManager struct {
	server *Server
	mu     sync.RWMutex
	configs map[string]*ConfigRecord // key: ne_id + config_type
}

// ConfigRecord 配置记录
type ConfigRecord struct {
	ID         string            `json:"id"`
	NEID       string            `json:"ne_id"`
	ConfigType string            `json:"config_type"`
	ConfigData []byte            `json:"config_data"`
	Version    int32             `json:"version"`
	CreatedAt  int64             `json:"created_at"` // 毫秒
	UpdatedAt  int64             `json:"updated_at"` // 毫秒
	ApplyTime  int64             `json:"apply_time"` // 毫秒
	Status     ConfigStatus      `json:"status"`
}

// ConfigStatus 配置状态
type ConfigStatus int

const (
	ConfigStatusPending ConfigStatus = iota
	ConfigStatusApplied
	ConfigStatusFailed
)

func NewConfigManager(srv *Server) *ConfigManager {
	return &ConfigManager{
		server:  srv,
		configs: make(map[string]*ConfigRecord),
	}
}

// ============================================
// 配置下发 PushConfig
// ============================================

func (cm *ConfigManager) PushConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	// 检查 NE 是否在线
	sessCtx := cm.server.SessionManager().Get(req.NeId)
	if sessCtx == nil {
		return &pb.ConfigResponse{
			Code:    1,
			Message: "NE not found or offline",
		}, nil
	}

	// 生成配置 ID
	configID := fmt.Sprintf("cfg-%s-%d", req.NeId, nowMs())

	// 存储配置记录
	key := req.NeId + ":" + req.ConfigType
	record := &ConfigRecord{
		ID:         configID,
		NEID:       req.NeId,
		ConfigType: req.ConfigType,
		ConfigData: req.ConfigData,
		Version:    req.Version,
		CreatedAt:  nowMs(),
		UpdatedAt:  nowMs(),
		ApplyTime:  nowMs(),
		Status:     ConfigStatusPending,
	}

	cm.mu.Lock()
	cm.configs[key] = record
	cm.mu.Unlock()

	// TODO: 将配置实际下发到 NE（通过已有的命令通道或专用配置通道）
	// 这里假设下发成功，实际应等待 NE 确认
	record.Status = ConfigStatusApplied

	return &pb.ConfigResponse{
		Code:     0,
		Message:  "success",
		ConfigId: configID,
		ApplyTime: record.ApplyTime,
	}, nil
}

// ============================================
// 获取配置 GetConfig
// ============================================

func (cm *ConfigManager) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.ConfigResponse, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := req.NeId + ":" + req.ConfigType
	if record, ok := cm.configs[key]; ok {
		return &pb.ConfigResponse{
			Code:     0,
			Message:  "success",
			ConfigId: record.ID,
			ApplyTime: record.ApplyTime,
		}, nil
	}

	return &pb.ConfigResponse{
		Code:    1,
		Message: "config not found",
	}, nil
}

// ============================================
// 配置同步 SyncConfig
// ============================================

func (cm *ConfigManager) SyncConfig(ctx context.Context, req *pb.SyncConfigRequest) (*pb.SyncConfigResponse, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := req.NeId + ":" + req.ConfigType
	if record, ok := cm.configs[key]; ok {
		// 检查版本
		if req.TargetVersion != "" && record.Version > 0 {
			// 返回当前版本配置
			return &pb.SyncConfigResponse{
				Code:           0,
				Message:        "success",
				ConfigData:     record.ConfigData,
				CurrentVersion: fmt.Sprintf("v%d", record.Version),
			}, nil
		}

		return &pb.SyncConfigResponse{
			Code:           0,
			Message:        "success",
			ConfigData:     record.ConfigData,
			CurrentVersion: fmt.Sprintf("v%d", record.Version),
		}, nil
	}

	return &pb.SyncConfigResponse{
		Code:    1,
		Message: "config not found",
	}, nil
}

// ============================================
// 查询 NE 的所有配置
// ============================================

func (cm *ConfigManager) ListConfigs(neID string) []*ConfigRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []*ConfigRecord
	for _, cfg := range cm.configs {
		if cfg.NEID == neID {
			results = append(results, cfg)
		}
	}
	return results
}

// ============================================
// 比较配置差异
// ============================================

func (cm *ConfigManager) CompareConfig(neID, configType string, newData []byte) *types.ConfigDiff {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := neID + ":" + configType
	record, ok := cm.configs[key]
	if !ok {
		return &types.ConfigDiff{
			NEID:       neID,
			ConfigType: configType,
			Added:      newData,
		}
	}

	// 简单比较（实际应该用更深层的 diff 算法）
	if string(record.ConfigData) == string(newData) {
		return nil
	}

	return &types.ConfigDiff{
		NEID:       neID,
		ConfigType: configType,
		Modified:   newData,
	}
}

// ============================================
// 辅助方法
// ============================================

func (cm *ConfigManager) exportJSON() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data := make(map[string]*ConfigRecord)
	for k, v := range cm.configs {
		data[k] = v
	}

	return json.MarshalIndent(data, "", "  ")
}