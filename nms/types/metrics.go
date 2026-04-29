package types

// ============================================
// 性能管理相关类型
// ============================================

// MetricsRequest 性能数据上报请求
type MetricsRequest struct {
	NEID        string        `json:"ne_id"`
	SessionID   string        `json:"session_id"`
	MetricsType string        `json:"metrics_type"` // 指标类型（如 cpu、memory、interface）
	Timestamp   int64         `json:"timestamp"`     // 毫秒
	Metrics     []*MetricItem `json:"metrics"`
}

// MetricItem 单个指标项
type MetricItem struct {
	Name   string            `json:"name"`   // 指标名称
	Value  string            `json:"value"`  // 指标值
	Unit   string            `json:"unit"`   // 单位
	Labels map[string]string `json:"labels"` // 标签
}

// MetricsResponse 性能数据响应
type MetricsResponse struct {
	NEID    string `json:"ne_id"`
	Ack     bool   `json:"ack"`
	Message string `json:"message"`
}

// MetricsQuery 性能数据查询条件
type MetricsQuery struct {
	NEID        string            `json:"ne_id"`
	MetricsType string            `json:"metrics_type"`
	StartTime   int64             `json:"start_time"` // 毫秒
	EndTime     int64             `json:"end_time"`   // 毫秒
	Names       []string          `json:"names"`      // 指标名称列表
	Labels      map[string]string `json:"labels"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
}

// MetricsRecord 性能数据记录（存储用）
type MetricsRecord struct {
	ID          string            `json:"id"`
	NEID        string            `json:"ne_id"`
	MetricsType string            `json:"metrics_type"`
	Name        string            `json:"name"`
	Value       string            `json:"value"`
	Unit        string            `json:"unit"`
	Labels      map[string]string `json:"labels"`
	Timestamp   int64             `json:"timestamp"` // 毫秒
	CreatedAt   int64             `json:"created_at"` // 毫秒
}

// MetricsHandler 性能数据处理回调函数类型
type MetricsHandler func(neID string, metrics []*MetricItem) error

// MetricAggregator 指标聚合器接口
type MetricAggregator interface {
	// Aggregate 聚合指标数据
	Aggregate(records []*MetricsRecord) (*AggregatedMetrics, error)
	// GroupBy 按标签分组
	GroupBy(records []*MetricsRecord, label string) (map[string][]*MetricsRecord, error)
}

// AggregatedMetrics 聚合后的指标数据
type AggregatedMetrics struct {
	NEID        string          `json:"ne_id"`
	MetricsType string          `json:"metrics_type"`
	Period      string          `json:"period"` // 聚合周期（1min、5min、1hour）
	Results     []*MetricResult `json:"results"`
}

// MetricResult 单个聚合结果
type MetricResult struct {
	Name     string            `json:"name"`
	Labels   map[string]string `json:"labels"`
	Min      float64           `json:"min"`
	Max      float64           `json:"max"`
	Avg      float64           `json:"avg"`
	Sum      float64           `json:"sum"`
	Count    int               `json:"count"`
}