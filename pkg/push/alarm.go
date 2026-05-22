package push

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

const (
	ALARM_TYPE_COMMUNICATION      = "CommunicationAlarm"    // 告警类型-通信警报
	ALARM_TYPE_EQUIPMENT          = "EquipmentAlarm"        // 告警类型-设备警报
	ALARM_TYPE_PROCESSING_FAILURE = "ProcessingFailure"     // 告警类型-处理故障
	ALARM_TYPE_ENVIRONMENTAL      = "EnvironmentalAlarm"    // 告警类型-环境警报
	ALARM_TYPE_QUALITY_OF_SERVICE = "QualityOfServiceAlarm" // 告警类型-服务质量警报
)

const (
	ALARM_SEVERITY_CRITICAL = "Critical" // 告警级别-危急
	ALARM_SEVERITY_MAJOR    = "Major"    // 告警级别-主要
	ALARM_SEVERITY_MINOR    = "Minor"    // 告警级别-次要
	ALARM_SEVERITY_WARNING  = "Warning"  // 告警级别-警告
	ALARM_SEVERITY_EVENT    = "Event"    // 告警级别-事件
)

const (
	ALARM_STATUS_CLEAR  = "Clear"  // 告警状态-清除
	ALARM_STATUS_ACTIVE = "Active" // 告警状态-活动
)

// ALARM_PUSH_URI 告警推送URI地址 POST
const ALARM_PUSH_URI = "/push/alarm/receive"

// Alarm 告警信息对象
type Alarm struct {
	NeUid             string `json:"neUid" binding:"required"`                                                                                                        // 网元唯一标识
	AlarmTime         int64  `json:"alarmTime" binding:"required"`                                                                                                    // 事件产生时间 时间戳毫秒，Push自动填充
	AlarmId           string `json:"alarmId" binding:"required"`                                                                                                      // 告警ID 唯一，清除时对应
	AlarmCode         int    `json:"alarmCode" binding:"required"`                                                                                                    // 告警状态码
	AlarmType         string `json:"alarmType" binding:"required,oneof=CommunicationAlarm EquipmentAlarm ProcessingFailure EnvironmentalAlarm QualityOfServiceAlarm"` // 告警类型 Communication Equipment ProcessingFailure Environmental QualityOfService
	AlarmTitle        string `json:"alarmTitle" binding:"required"`                                                                                                   // 告警标题
	PerceivedSeverity string `json:"perceivedSeverity" binding:"required,oneof=Critical Major Minor Warning Event"`                                                   // 告警级别 Critical,Major,Minor,Warning,Event
	AlarmStatus       string `json:"alarmStatus" binding:"required,oneof=Clear Active"`                                                                               // 告警状态 Clear,Active
	SpecificProblem   string `json:"specificProblem"`                                                                                                                 // 告警问题原因
	SpecificProblemID string `json:"specificProblemId"`                                                                                                               // 告警问题原因ID
	AddInfo           string `json:"addInfo"`                                                                                                                         // 告警辅助信息
	LocationInfo      string `json:"locationInfo"`                                                                                                                    // 告警定位信息
}

// AlarmService 告警服务
type AlarmService struct {
	alarmHistorys        *ringbuffer.RingBuffer[Alarm] // 告警历史记录（环形缓冲区）
	alarmHistorysMaxSize atomic.Int32                  // 最大历史记录数量
}

// NewAlarmService 创建告警服务
func NewAlarmService() *AlarmService {
	a := &AlarmService{
		alarmHistorys: ringbuffer.NewRingBuffer[Alarm](4096),
	}
	a.alarmHistorysMaxSize.Store(4096)
	return a
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *AlarmService) HistoryList(n int) []Alarm {
	if s == nil {
		return []Alarm{}
	}
	if n < 0 {
		return []Alarm{}
	}
	if n == 0 {
		return s.alarmHistorys.GetAll()
	}
	return s.alarmHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
func (s *AlarmService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.alarmHistorysMaxSize.Store(int32(newSize))
	s.alarmHistorys.Resize(newSize)
}

// PushURL 告警推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *AlarmService) PushURL(url string, alarm *Alarm, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	alarm.AlarmTime = time.Now().UnixMilli()
	s.alarmHistorys.Push(*alarm)
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, alarm)
}
