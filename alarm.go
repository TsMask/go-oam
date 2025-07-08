package oam

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/push/model"
	"github.com/tsmask/go-oam/src/modules/push/service"

	"github.com/gin-gonic/gin"
)

const (
	AlarmTypeCommunicationAlarm    = "CommunicationAlarm"    // 告警类型-通信警报
	AlarmTypeEquipmentAlarm        = "EquipmentAlarm"        // 告警类型-设备警报
	AlarmTypeProcessingFailure     = "ProcessingFailure"     // 告警类型-处理故障
	AlarmTypeEnvironmentalAlarm    = "EnvironmentalAlarm"    // 告警类型-环境警报
	AlarmTypeQualityOfServiceAlarm = "QualityOfServiceAlarm" // 告警类型-服务质量警报
)

const (
	AlarmSeverityCritical = "Critical" // 告警级别-危急
	AlarmSeverityMajor    = "Major"    // 告警级别-主要
	AlarmSeverityMinor    = "Minor"    // 告警级别-次要
	AlarmSeverityWarning  = "Warning"  // 告警级别-警告
	AlarmSeverityEvent    = "Event"    // 告警级别-事件
)

const (
	AlarmStatusClear  = "Clear"  // 告警状态-清除
	AlarmStatusActive = "Active" // 告警状态-活动
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
func AlarmReceiveRoute(router gin.IRouter, onReceive func(Alarm)) {
	router.POST(service.ALARM_PUSH_URI, func(c *gin.Context) {
		var body Alarm
		if err := c.ShouldBindBodyWithJSON(&body); err != nil {
			errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
			c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
			return
		}
		onReceive(body)
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
