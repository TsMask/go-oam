package model

const (
	ALARM_TYPE_COMMUNICATION_ALARM      = "CommunicationAlarm"    // 告警类型-通信警报
	ALARM_TYPE_EQUIPMENT_ALARM          = "EquipmentAlarm"        // 告警类型-设备警报
	ALARM_TYPE_PROCESSING_FAILURE       = "ProcessingFailure"     // 告警类型-处理故障
	ALARM_TYPE_ENVIRONMENTAL_ALARM      = "EnvironmentalAlarm"    // 告警类型-环境警报
	ALARM_TYPE_QUALITY_OF_SERVICE_ALARM = "QualityOfServiceAlarm" // 告警类型-服务质量警报
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

// Alarm 告警信息对象
type Alarm struct {
	NeUid             string `json:"neUid" binding:"required"`             // 网元唯一标识
	AlarmTime         int64  `json:"alarmTime" binding:"required"`         // 事件产生时间 时间戳毫秒，Push自动填充
	AlarmId           string `json:"alarmId" binding:"required"`           // 告警ID 唯一，清除时对应
	AlarmCode         int    `json:"alarmCode" binding:"required"`         // 告警状态码
	AlarmType         string `json:"alarmType" binding:"required"`         // 告警类型 CommunicationAlarm,EquipmentAlarm,ProcessingFailure,EnvironmentalAlarm,QualityOfServiceAlarm
	AlarmTitle        string `json:"alarmTitle" binding:"required"`        // 告警标题
	PerceivedSeverity string `json:"perceivedSeverity" binding:"required"` // 告警级别 Critical,Major,Minor,Warning,Event
	AlarmStatus       string `json:"alarmStatus" binding:"required"`       // 告警状态 Clear,Active
	SpecificProblem   string `json:"specificProblem"`                      // 告警问题原因
	SpecificProblemID string `json:"specificProblemId"`                    // 告警问题原因ID
	AddInfo           string `json:"addInfo"`                              // 告警辅助信息
	LocationInfo      string `json:"locationInfo"`                         // 告警定位信息
}
