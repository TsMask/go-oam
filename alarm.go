package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/push/model"
	"github.com/tsmask/go-oam/src/modules/push/service"

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

// AlarmHistoryList 告警历史列表
// 需要先调用 AlarmHistoryClearTimer 才会开启记录
func AlarmHistoryList() []Alarm {
	return service.AlarmHistoryList()
}

// AlarmHistoryClearTimer 告警历史定时清除 数据保留一天，0点重新记录
func AlarmHistoryClearTimer() {
	service.AlarmHistoryClearTimer()
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

// AlarmPushURL 告警推送 自定义URL地址接收
func AlarmPushURL(url string, alarm *Alarm) error {
	return service.AlarmPushURL(url, alarm)
}

// AlarmPush 告警推送
// 默认URL地址：ALARM_PUSH_URI
//
// protocol 协议 http(s)
//
// host 服务地址 如：192.168.5.58:33020
func AlarmPush(protocol, host string, alarm *Alarm) error {
	url := fmt.Sprintf("%s://%s%s", protocol, host, service.ALARM_PUSH_URI)
	return service.AlarmPushURL(url, alarm)
}
