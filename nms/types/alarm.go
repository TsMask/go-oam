package types

// ============================================
// 告警相关类型
// ============================================

// AlarmData 告警数据（内部使用）
type AlarmData struct {
	AlarmID     string            `json:"alarm_id"`      // 告警唯一 ID
	AlarmType   string            `json:"alarm_type"`     // 告警类型
	Severity    string            `json:"severity"`        // critical/major/minor/warning/info
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Source      string            `json:"source"`         // 告警源
	StartTime   int64             `json:"start_time"`     // 毫秒
	EndTime     int64             `json:"end_time"`       // 毫秒，0 表示未结束
	Params      map[string]string `json:"params"`         // 扩展参数

	// 内部字段
	NEID       string `json:"ne_id"`
	SessionID  string `json:"session_id"`
	ReceivedAt int64  `json:"received_at"` // 毫秒
}

// AlarmRequest 告警上报请求
type AlarmRequest struct {
	NEID      string    `json:"ne_id"`
	SessionID string    `json:"session_id"`
	Alarm     *AlarmData `json:"alarm"`
}

// AlarmResponse 告警响应
type AlarmResponse struct {
	NEID    string `json:"ne_id"`
	AlarmID string `json:"alarm_id"`
	Ack     bool   `json:"ack"`
	Message string `json:"message"`
}

// AlarmSeverity 告警严重程度常量
var AlarmSeverity = struct {
	Critical string
	Major    string
	Minor    string
	Warning  string
	Info     string
}{
	Critical: "critical",
	Major:    "major",
	Minor:    "minor",
	Warning:  "warning",
	Info:     "info",
}

// AlarmFilter 告警过滤条件
type AlarmFilter struct {
	NEID      string `json:"ne_id"`
	AlarmType string `json:"alarm_type"`
	Severity  string `json:"severity"`
	StartTime int64  `json:"start_time"` // 毫秒
	EndTime   int64  `json:"end_time"`   // 毫秒
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

// AlarmStatistic 告警统计
type AlarmStatistic struct {
	NEID          string `json:"ne_id"`
	CriticalCount int    `json:"critical_count"`
	MajorCount    int    `json:"major_count"`
	MinorCount    int    `json:"minor_count"`
	WarningCount  int    `json:"warning_count"`
	InfoCount     int    `json:"info_count"`
	TotalCount    int    `json:"total_count"`
}

// AlarmHandler 告警处理回调函数类型
type AlarmHandler func(alarm *AlarmData) error