package types

// ============================================
// 配置管理相关类型
// ============================================

// ConfigRequest 配置下发请求
type ConfigRequest struct {
	NEID       string `json:"ne_id"`
	SessionID  string `json:"session_id"`
	ConfigType string `json:"config_type"` // 配置类型（如 interface、route、acl）
	ConfigData []byte `json:"config_data"` // 配置内容
	Version    int32  `json:"version"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	Code      int32 `json:"code"`
	Message   string `json:"message"`
	ConfigID  string `json:"config_id"` // 配置记录 ID
	ApplyTime int64  `json:"apply_time"` // 毫秒
}

// GetConfigRequest 获取配置请求
type GetConfigRequest struct {
	NEID       string `json:"ne_id"`
	SessionID  string `json:"session_id"`
	ConfigType string `json:"config_type"`
}

// SyncConfigRequest 同步配置请求
type SyncConfigRequest struct {
	NEID          string `json:"ne_id"`
	SessionID     string `json:"session_id"`
	ConfigType    string `json:"config_type"`
	TargetVersion string `json:"target_version"`
}

// SyncConfigResponse 同步配置响应
type SyncConfigResponse struct {
	Code           int32  `json:"code"`
	Message        string `json:"message"`
	ConfigData     []byte `json:"config_data"`
	CurrentVersion string `json:"current_version"`
}

// ConfigRecord 配置记录（存储用）
type ConfigRecord struct {
	ID         string `json:"id"`
	NEID       string `json:"ne_id"`
	ConfigType string `json:"config_type"`
	ConfigData []byte `json:"config_data"`
	Version    int32  `json:"version"`
	CreatedAt  int64  `json:"created_at"` // 毫秒
	UpdatedAt  int64  `json:"updated_at"` // 毫秒
}

// ConfigDiff 配置差异（比较用）
type ConfigDiff struct {
	NEID       string   `json:"ne_id"`
	ConfigType string   `json:"config_type"`
	Added      []byte   `json:"added"`
	Removed    []byte   `json:"removed"`
	Modified   []byte   `json:"modified"`
}