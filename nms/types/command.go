package types

// ============================================
// 命令控制相关类型
// ============================================

// CommandRequest 命令执行请求
type CommandRequest struct {
	NEID        string            `json:"ne_id"`
	SessionID   string            `json:"session_id"`
	Command     string            `json:"command"`      // 命令内容
	CommandType string            `json:"command_type"` // cli/snmp/netconf
	Timeout     int32             `json:"timeout"`     // 超时时间（秒）
	Params      map[string]string `json:"params"`      // 扩展参数
}

// CommandResponse 命令执行响应
type CommandResponse struct {
	Code     int32  `json:"code"`
	Message  string `json:"message"`
	Result   string `json:"result"`    // 命令执行结果
	ExecTime int64  `json:"exec_time"` // 毫秒
}

// BatchCommandRequest 批量命令执行请求
type BatchCommandRequest struct {
	NEIDs      []string         `json:"ne_ids"`
	SessionID  string           `json:"session_id"`
	Command    string           `json:"command"`
	CommandType string          `json:"command_type"`
	Timeout    int32            `json:"timeout"`
	Params     map[string]string `json:"params"`
}

// BatchCommandResponse 批量命令执行响应（流式）
type BatchCommandResponse struct {
	NEID     string `json:"ne_id"`
	Code     int32  `json:"code"`
	Message  string `json:"message"`
	Result   string `json:"result"`
	ExecTime int64  `json:"exec_time"` // 毫秒
	Done     bool   `json:"done"` // 是否全部完成
}

// CommandType 命令类型常量
var CommandType = struct {
	CLI     string
	SNMP    string
	NETCONF string
}{
	CLI:     "cli",
	SNMP:    "snmp",
	NETCONF: "netconf",
}

// CommandResult 命令执行结果（存储用）
type CommandResult struct {
	ID           string `json:"id"`
	NEID         string `json:"ne_id"`
	Command      string `json:"command"`
	CommandType  string `json:"command_type"`
	Code         int32  `json:"code"`
	Message      string `json:"message"`
	Result       string `json:"result"`
	ExecTime     int64  `json:"exec_time"` // 毫秒
	CreatedAt    int64  `json:"created_at"` // 毫秒
}

// CommandHandler 命令处理回调函数类型
type CommandHandler func(neID string, req *CommandRequest) (*CommandResponse, error)

// CommandHistory 命令历史记录
type CommandHistory struct {
	ID          string `json:"id"`
	NEID        string `json:"ne_id"`
	Command     string `json:"command"`
	CommandType string `json:"command_type"`
	Code        int32  `json:"code"`
	Result      string `json:"result"`
	ExecTime    int64  `json:"exec_time"` // 毫秒
	ExecutedBy  string `json:"executed_by"` // 执行人
	ExecutedAt  int64  `json:"executed_at"` // 毫秒
}

// CommandTemplate 命令模板
type CommandTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     string   `json:"command"`
	CommandType string   `json:"command_type"`
	Params      []string `json:"params"` // 参数名列表
}

// CommandExecutor 命令执行器接口
type CommandExecutor interface {
	// Execute 执行单条命令
	Execute(neID string, req *CommandRequest) (*CommandResponse, error)
	// BatchExecute 批量执行命令
	BatchExecute(neID string, req *BatchCommandRequest) (<-chan *BatchCommandResponse, error)
}