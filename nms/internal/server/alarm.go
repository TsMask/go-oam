package server

import (
	"fmt"
	"sync"

	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// AlarmManager 告警管理器
// ============================================

type AlarmManager struct {
	server  *Server
	mu      sync.RWMutex
	alarms  map[string]*types.AlarmData // key: alarm_id
	filters []AlarmFilter              // 告警过滤器
}

// AlarmFilter 告警过滤器
type AlarmFilter struct {
	NEID      string   // NE ID（空表示所有）
	Types     []string // 告警类型列表
	Severities []string // 严重程度列表
}

// NewAlarmManager 创建告警管理器
func NewAlarmManager(srv *Server) *AlarmManager {
	return &AlarmManager{
		server: srv,
		alarms: make(map[string]*types.AlarmData),
	}
}

// ============================================
// 处理上报的告警
// ============================================

func (am *AlarmManager) HandleAlarm(neID string, alarm *types.AlarmData) error {
	// 设置 NE ID 和接收时间
	alarm.NEID = neID
	alarm.ReceivedAt = nowMs()

	// 检查过滤器
	if !am.matchFilter(alarm) {
		return fmt.Errorf("alarm filtered")
	}

	am.mu.Lock()
	am.alarms[alarm.AlarmID] = alarm
	am.mu.Unlock()

	// TODO: 触发告警通知（如邮件、短信、webhook）
	fmt.Printf("[Alarm] NE=%s Type=%s Severity=%s Name=%s\n",
		alarm.NEID, alarm.AlarmType, alarm.Severity, alarm.Name)

	return nil
}

// matchFilter 检查告警是否匹配过滤器
func (am *AlarmManager) matchFilter(alarm *types.AlarmData) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, f := range am.filters {
		// 检查 NE ID
		if f.NEID != "" && f.NEID != alarm.NEID {
			continue
		}

		// 检查告警类型
		if len(f.Types) > 0 {
			match := false
			for _, t := range f.Types {
				if t == alarm.AlarmType {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// 检查严重程度
		if len(f.Severities) > 0 {
			match := false
			for _, s := range f.Severities {
				if s == alarm.Severity {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// 匹配到过滤器
		return true
	}

	// 没有过滤器或没有匹配到，返回 true
	return len(am.filters) == 0
}

// AddFilter 添加过滤器
func (am *AlarmManager) AddFilter(f AlarmFilter) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.filters = append(am.filters, f)
}

// ============================================
// 告警确认 AckAlarm
// ============================================

func (am *AlarmManager) AckAlarm(alarmID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if alarm, ok := am.alarms[alarmID]; ok {
		alarm.EndTime = nowMs() // 标记告警结束
		return nil
	}

	return fmt.Errorf("alarm not found: %s", alarmID)
}

// ============================================
// 查询告警列表
// ============================================

func (am *AlarmManager) QueryAlarms(q *types.AlarmFilter) []*types.AlarmData {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var results []*types.AlarmData
	for _, alarm := range am.alarms {
		// 检查 NE ID
		if q.NEID != "" && alarm.NEID != q.NEID {
			continue
		}

		// 检查告警类型
		if q.AlarmType != "" && alarm.AlarmType != q.AlarmType {
			continue
		}

		// 检查严重程度
		if q.Severity != "" && alarm.Severity != q.Severity {
			continue
		}

		// 检查时间范围
		if q.StartTime > 0 && alarm.StartTime < q.StartTime {
			continue
		}
		if q.EndTime > 0 && alarm.EndTime > q.EndTime {
			continue
		}

		results = append(results, alarm)
	}

	return results
}

// ============================================
// 获取告警统计
// ============================================

func (am *AlarmManager) GetStatistics(neID string) *types.AlarmStatistic {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := &types.AlarmStatistic{NEID: neID}

	for _, alarm := range am.alarms {
		if neID != "" && alarm.NEID != neID {
			continue
		}
		// 只统计未结束的告警
		if alarm.EndTime == 0 {
			stats.TotalCount++
			switch alarm.Severity {
			case types.AlarmSeverity.Critical:
				stats.CriticalCount++
			case types.AlarmSeverity.Major:
				stats.MajorCount++
			case types.AlarmSeverity.Minor:
				stats.MinorCount++
			case types.AlarmSeverity.Warning:
				stats.WarningCount++
			case types.AlarmSeverity.Info:
				stats.InfoCount++
			}
		}
	}

	return stats
}

// ============================================
// 获取当前活跃的告警数
// ============================================

func (am *AlarmManager) GetActiveCount(neID string) int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	count := 0
	for _, alarm := range am.alarms {
		if alarm.EndTime == 0 && (neID == "" || alarm.NEID == neID) {
			count++
		}
	}
	return count
}

// ============================================
// 清除历史告警（超过一定时间的）
// ============================================

func (am *AlarmManager) Cleanup(maxAgeMs int64) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := nowMs()
	removed := 0

	for id, alarm := range am.alarms {
		// 只清理已结束的告警
		if alarm.EndTime > 0 && now-alarm.EndTime > maxAgeMs {
			delete(am.alarms, id)
			removed++
		}
	}

	return removed
}