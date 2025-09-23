package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"

	"github.com/gin-gonic/gin"
)

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

type Alarm = model.Alarm

// AlarmPush 告警推送
// 默认URL地址：ALARM_PUSH_URI
func AlarmPush(alarm *Alarm) error {
	omcInfo := OMCInfoGet()
	if omcInfo.Url == "" {
		return fmt.Errorf("omc url is empty")
	}
	url := fmt.Sprintf("%s%s", omcInfo.Url, service.ALARM_PUSH_URI)
	alarm.NeUid = omcInfo.NeUID
	return service.AlarmPushURL(url, alarm)
}

// AlarmPushURL 告警推送 自定义URL地址接收
func AlarmPushURL(url string, alarm *Alarm) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	return service.AlarmPushURL(url, alarm)
}

// AlarmHistoryList 告警历史列表
// n 为返回的最大记录数，n<0返回空列表
func AlarmHistoryList(n int) []Alarm {
	return service.AlarmHistoryList(n)
}

// AlarmHistorySetSize 设置告警历史列表数量
// 当设置的大小小于当前历史记录数时，会自动清理旧记录
// 默认值 4096
func AlarmHistorySetSize(size int) {
	service.AlarmHistorySetSize(size)
}

// AlarmReceiveRoute 告警接收路由装载
// 接收端实现
func AlarmReceiveRoute(router gin.IRouter, onReceive func(Alarm) error) {
	router.POST(service.ALARM_PUSH_URI, func(c *gin.Context) {
		var body Alarm
		if err := c.ShouldBindBodyWithJSON(&body); err != nil {
			errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
			c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
			return
		}
		if err := onReceive(body); err != nil {
			c.JSON(200, resp.ErrMsg(err.Error()))
			return
		}
		c.JSON(200, resp.Ok(nil))
	})
}
