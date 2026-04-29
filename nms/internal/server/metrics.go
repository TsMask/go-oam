package server

import (
	"fmt"
	"sync"

	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// MetricsManager 性能数据管理器
// ============================================

type MetricsManager struct {
	server *Server
	mu     sync.RWMutex

	// 实时指标（最近的数据点）
	realtimeMetrics map[string]map[string]string // key: neID + ":" + metricsType -> key: metricName -> value

	// 历史数据（可按需扩展为时序数据库）
	historyMetrics []*types.MetricsRecord

	// 告警阈值配置
	thresholds map[string]*ThresholdConfig // key: neID + ":" + metricName
}

// ThresholdConfig 阈值配置
type ThresholdConfig struct {
	NEID         string
	MetricName   string
	WarningValue float64
	CriticalValue float64
	Operator     string // ">=" / "<=" / "==" / ">"
}

// NewMetricsManager 创建性能管理器
func NewMetricsManager(srv *Server) *MetricsManager {
	return &MetricsManager{
		server:         srv,
		realtimeMetrics: make(map[string]map[string]string),
		historyMetrics:  make([]*types.MetricsRecord, 0, 10000),
		thresholds:     make(map[string]*ThresholdConfig),
	}
}

// ============================================
// 处理上报的性能数据
// ============================================

func (mm *MetricsManager) HandleMetrics(neID string, metricsType string, items []*types.MetricItem) error {
	now := nowMs()

	// 存储实时指标
	mm.mu.Lock()
	key := neID + ":" + metricsType

	// 初始化或获取该 NE 的指标映射
	if mm.realtimeMetrics[key] == nil {
		mm.realtimeMetrics[key] = make(map[string]string)
	}

	// 更新实时指标
	for _, item := range items {
		mm.realtimeMetrics[key][item.Name] = item.Value

		// 检查阈值
		mm.checkThreshold(neID, item)

		// 存储历史记录
		record := &types.MetricsRecord{
			ID:          fmt.Sprintf("m-%d", now),
			NEID:        neID,
			MetricsType: metricsType,
			Name:        item.Name,
			Value:       item.Value,
			Unit:        item.Unit,
			Labels:      item.Labels,
			Timestamp:   now,
			CreatedAt:  now,
		}
		mm.historyMetrics = append(mm.historyMetrics, record)
	}

	mm.mu.Unlock()

	// 打印日志
	fmt.Printf("[Metrics] NE=%s Type=%s Count=%d\n", neID, metricsType, len(items))

	return nil
}

// ============================================
// 检查阈值
// ============================================

func (mm *MetricsManager) checkThreshold(neID string, item *types.MetricItem) {
	thresholdKey := neID + ":" + item.Name

	mm.mu.RLock()
	threshold, ok := mm.thresholds[thresholdKey]
	mm.mu.RUnlock()

	if !ok {
		return
	}

	// 解析当前值
	var currentValue float64
	fmt.Sscanf(item.Value, "%f", &currentValue)

	// 检查阈值
	if currentValue >= threshold.CriticalValue {
		// 触发严重告警
		alarm := &types.AlarmData{
			AlarmID:   fmt.Sprintf("alarm-threshold-%d", nowMs()),
			AlarmType: "threshold_exceeded",
			Severity:  types.AlarmSeverity.Critical,
			Name:      fmt.Sprintf("%s Critical Threshold Exceeded", item.Name),
			Description: fmt.Sprintf("%s = %s (critical threshold: %.2f)",
				item.Name, item.Value, threshold.CriticalValue),
			Source:    neID,
			StartTime: nowMs(),
		}

		// 通知告警管理器
		if am := mm.server.alarmMgr; am != nil {
			am.HandleAlarm(neID, alarm)
		}
	} else if currentValue >= threshold.WarningValue {
		// 触发警告告警
		alarm := &types.AlarmData{
			AlarmID:   fmt.Sprintf("alarm-threshold-%d", nowMs()),
			AlarmType: "threshold_warning",
			Severity:  types.AlarmSeverity.Warning,
			Name:      fmt.Sprintf("%s Warning Threshold Exceeded", item.Name),
			Description: fmt.Sprintf("%s = %s (warning threshold: %.2f)",
				item.Name, item.Value, threshold.WarningValue),
			Source:    neID,
			StartTime: nowMs(),
		}

		if am := mm.server.alarmMgr; am != nil {
			am.HandleAlarm(neID, alarm)
		}
	}
}

// SetThreshold 设置阈值
func (mm *MetricsManager) SetThreshold(neID, metricName string, warning, critical float64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	key := neID + ":" + metricName
	mm.thresholds[key] = &ThresholdConfig{
		NEID:          neID,
		MetricName:    metricName,
		WarningValue:  warning,
		CriticalValue: critical,
	}
}

// ============================================
// 查询实时指标
// ============================================

func (mm *MetricsManager) GetRealtime(neID, metricsType string) map[string]string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	key := neID + ":" + metricsType
	if metrics, ok := mm.realtimeMetrics[key]; ok {
		result := make(map[string]string)
		for k, v := range metrics {
			result[k] = v
		}
		return result
	}
	return nil
}

// ============================================
// 查询历史数据
// ============================================

func (mm *MetricsManager) QueryHistory(q *types.MetricsQuery) []*types.MetricsRecord {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var results []*types.MetricsRecord
	for _, record := range mm.historyMetrics {
		// 检查 NE ID
		if q.NEID != "" && record.NEID != q.NEID {
			continue
		}

		// 检查指标类型
		if q.MetricsType != "" && record.MetricsType != q.MetricsType {
			continue
		}

		// 检查时间范围
		if q.StartTime > 0 && record.Timestamp < q.StartTime {
			continue
		}
		if q.EndTime > 0 && record.Timestamp > q.EndTime {
			continue
		}

		// 检查指标名称
		if len(q.Names) > 0 {
			match := false
			for _, name := range q.Names {
				if name == record.Name {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		results = append(results, record)
	}

	// 分页
	if q.Page > 0 && q.PageSize > 0 {
		start := (q.Page - 1) * q.PageSize
		end := start + q.PageSize
		if start >= len(results) {
			return nil
		}
		if end > len(results) {
			end = len(results)
		}
		results = results[start:end]
	}

	return results
}

// ============================================
// 聚合数据
// ============================================

func (mm *MetricsManager) Aggregate(neID, metricsType, name string, intervalMs int64) (*types.AggregatedMetrics, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// 收集数据点
	var values []float64
	for _, record := range mm.historyMetrics {
		if record.NEID != neID {
			continue
		}
		if metricsType != "" && record.MetricsType != metricsType {
			continue
		}
		if name != "" && record.Name != name {
			continue
		}

		var val float64
		fmt.Sscanf(record.Value, "%f", &val)
		values = append(values, val)
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no data for aggregation")
	}

	// 计算聚合值
	var min, max, sum float64
	min = values[0]
	max = values[0]
	sum = values[0]

	for i := 1; i < len(values); i++ {
		if values[i] < min {
			min = values[i]
		}
		if values[i] > max {
			max = values[i]
		}
		sum += values[i]
	}

	avg := sum / float64(len(values))

	return &types.AggregatedMetrics{
		NEID:        neID,
		MetricsType: metricsType,
		Period:      fmt.Sprintf("%dms", intervalMs),
		Results: []*types.MetricResult{
			{Name: name, Min: min, Max: max, Avg: avg, Sum: sum, Count: len(values)},
		},
	}, nil
}

// ============================================
// 清理历史数据
// ============================================

func (mm *MetricsManager) Cleanup(maxRecords int) int {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if len(mm.historyMetrics) <= maxRecords {
		return 0
	}

	removed := len(mm.historyMetrics) - maxRecords
	mm.historyMetrics = mm.historyMetrics[removed:]

	return removed
}