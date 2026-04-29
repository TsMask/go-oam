package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/tsmask/go-oam/nms/proto"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// CommandManager 命令控制器
// ============================================

type CommandManager struct {
	server *Server
	mu     sync.RWMutex

	// 命令执行器（可插拔）
	executors map[string]CommandExecutor // key: command_type

	// 历史记录
	history []*types.CommandResult

	// 模板管理
	templates map[string]*types.CommandTemplate
}

// CommandExecutor 命令执行器接口
type CommandExecutor interface {
	Execute(neID string, command string, timeout int32) (*CommandResponse, error)
}

// ============================================
// 默认执行器（通过 gRPC 发送到 NE）
// ============================================

type defaultExecutor struct {
	server *Server
}

func (e *defaultExecutor) Execute(neID string, command string, timeout int32) (*CommandResponse, error) {
	// 这里应该通过会话发送到 NE
	// 简化实现：模拟执行
	start := nowMs()

	// 模拟命令执行
	time.Sleep(100 * time.Millisecond)

	return &CommandResponse{
		Code:     0,
		Message:  "success",
		Result:   fmt.Sprintf("executed on %s: %s", neID, command),
		ExecTime: nowMs() - start,
	}, nil
}

// ============================================
// 构造函数
// ============================================

func NewCommandManager(srv *Server) *CommandManager {
	cm := &CommandManager{
		server:    srv,
		executors: make(map[string]CommandExecutor),
		templates: make(map[string]*types.CommandTemplate),
		history:   make([]*types.CommandResult, 0, 1000),
	}

	// 注册默认执行器
	cm.RegisterExecutor(types.CommandType.CLI, &defaultExecutor{server: srv})
	cm.RegisterExecutor(types.CommandType.SNMP, &defaultExecutor{server: srv})
	cm.RegisterExecutor(types.CommandType.NETCONF, &defaultExecutor{server: srv})

	return cm
}

// ============================================
// 注册执行器
// ============================================

func (cm *CommandManager) RegisterExecutor(commandType string, executor CommandExecutor) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.executors[commandType] = executor
}

// ============================================
// 执行单条命令
// ============================================

func (cm *CommandManager) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	start := nowMs()

	// 获取执行器
	cm.mu.RLock()
	executor, ok := cm.executors[req.CommandType]
	cm.mu.RUnlock()

	if !ok {
		// 使用默认执行器
		executor = &defaultExecutor{server: cm.server}
	}

	// 执行命令
	resp, err := executor.Execute(req.NeId, req.Command, req.Timeout)
	if err != nil {
		return &pb.CommandResponse{
			Code:     1,
			Message:  err.Error(),
			Result:   "",
			ExecTime: nowMs() - start,
		}, nil
	}

	// 记录历史
	cm.addHistory(req.NeId, req.Command, req.CommandType, resp)

	return &pb.CommandResponse{
		Code:     resp.Code,
		Message:  resp.Message,
		Result:   resp.Result,
		ExecTime: resp.ExecTime,
	}, nil
}

// ============================================
// 批量执行命令
// ============================================

func (cm *CommandManager) BatchExecuteCommand(req *pb.BatchCommandRequest, stream pb.NMSService_BatchExecuteCommandServer) error {
	cm.mu.RLock()
	executor, ok := cm.executors[req.CommandType]
	if !ok {
		executor = &defaultExecutor{server: cm.server}
	}
	cm.mu.RUnlock()

	for _, neID := range req.NeIds {
		start := nowMs()

		resp, err := executor.Execute(neID, req.Command, req.Timeout)
		if err != nil {
			stream.Send(&pb.CommandResponse{
				Code:     1,
				Message:  err.Error(),
				Result:   "",
				ExecTime: nowMs() - start,
				NeId:     neID,
			})
			continue
		}

		// 记录历史
		cm.addHistory(neID, req.Command, req.CommandType, resp)

		// 发送结果
		if err := stream.Send(&pb.CommandResponse{
			Code:     resp.Code,
			Message:  resp.Message,
			Result:   resp.Result,
			ExecTime: resp.ExecTime,
			NeId:     neID,
		}); err != nil {
			return err
		}
	}

	return nil
}

// ============================================
// 历史记录管理
// ============================================

func (cm *CommandManager) addHistory(neID, command, commandType string, resp *CommandResponse) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	result := &types.CommandResult{
		ID:          fmt.Sprintf("cmd-%d", nowMs()),
		NEID:        neID,
		Command:     command,
		CommandType: commandType,
		Code:        resp.Code,
		Message:     resp.Message,
		Result:      resp.Result,
		ExecTime:    resp.ExecTime,
		CreatedAt:   nowMs(),
	}

	cm.history = append(cm.history, result)

	// 限制历史记录数量
	if len(cm.history) > 10000 {
		cm.history = cm.history[1000:] // 保留最近 1000 条
	}
}

// QueryHistory 查询命令历史
func (cm *CommandManager) QueryHistory(neID string, limit int) []*types.CommandResult {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []*types.CommandResult
	for i := len(cm.history) - 1; i >= 0 && len(results) < limit; i-- {
		if neID == "" || cm.history[i].NEID == neID {
			results = append(results, cm.history[i])
		}
	}

	return results
}

// ============================================
// 模板管理
// ============================================

// RegisterTemplate 注册命令模板
func (cm *CommandManager) RegisterTemplate(tmpl *types.CommandTemplate) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.templates[tmpl.ID] = tmpl
}

// GetTemplate 获取命令模板
func (cm *CommandManager) GetTemplate(id string) *types.CommandTemplate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.templates[id]
}

// ListTemplates 列出所有模板
func (cm *CommandManager) ListTemplates() []*types.CommandTemplate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []*types.CommandTemplate
	for _, tmpl := range cm.templates {
		results = append(results, tmpl)
	}
	return results
}

// ExecuteTemplate 执行模板
func (cm *CommandManager) ExecuteTemplate(neID, templateID string, params map[string]string) (*CommandResponse, error) {
	tmpl := cm.GetTemplate(templateID)
	if tmpl == nil {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// 替换参数
	command := tmpl.Command
	for k, v := range params {
		command = replaceAll(command, "{{"+k+"}}", v)
	}

	// 执行
	cm.mu.RLock()
	executor, ok := cm.executors[tmpl.CommandType]
	if !ok {
		executor = &defaultExecutor{server: cm.server}
	}
	cm.mu.RUnlock()

	start := nowMs()
	resp, err := executor.Execute(neID, command, 30)
	if err != nil {
		return &CommandResponse{
			Code:    1,
			Message: err.Error(),
		}, nil
	}

	return &CommandResponse{
		Code:     resp.Code,
		Message:  resp.Message,
		Result:   resp.Result,
		ExecTime: nowMs() - start,
	}, nil
}

// ============================================
// 辅助函数
// ============================================

func replaceAll(s, old, new string) string {
	result := s
	for {
		idx := -1
		for i := 0; i <= len(result)-len(old); i++ {
			if result[i:i+len(old)] == old {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		result = result[:idx] + new + result[idx+len(old):]
	}
	return result
}

// CommandResponse 命令响应（内部使用）
type CommandResponse struct {
	Code     int32
	Message  string
	Result   string
	ExecTime int64
}