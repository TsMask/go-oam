package service

import (
	"time"

	"github.com/tsmask/go-oam/src/framework/fetch"
	"github.com/tsmask/go-oam/src/modules/push/model"
)

// ALARM_PUSH_URI 告警推送URI地址 POST
const ALARM_PUSH_URI = "/push/alarm/receive"

// 告警序号 每次发送时进行累加，0点重新记录
var alarmSeq int64 = 0

// alarmRecord 控制是否记录历史告警
var alarmRecord bool = false

// alarmHistorys 告警历史 数据保留一天，0点重新记录
var alarmHistorys []model.Alarm = make([]model.Alarm, 0)

// alarmClearDuration 计算到下一个时间间隔
func alarmClearDuration() time.Duration {
	now := time.Now()
	// 午夜（0点）
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	return midnight.Sub(now)
}

// AlarmHistoryClearTimer 历史清除
func AlarmHistoryClearTimer() {
	// 启动时允许记录历史告警
	alarmRecord = true
	// 创建一个定时器，在计算出的时间后触发第一次执行
	timer := time.NewTimer(alarmClearDuration())

	go func() {
		for {
			<-timer.C
			// 执行清除操作
			alarmHistorys = make([]model.Alarm, 0)
			// 重置定时器
			timer.Reset(alarmClearDuration())
		}
	}()
}

// AlarmHistoryList 历史列表
func AlarmHistoryList() []model.Alarm {
	return alarmHistorys
}

// AlarmPushURL 告警推送 自定义URL地址接收
func AlarmPushURL(url string, alarm *model.Alarm) error {
	alarmSeq++
	alarm.AlarmTime = time.Now().UnixMilli()
	alarm.AlarmSeq = alarmSeq
	// 发送
	_, err := fetch.PostJSON(url, alarm, nil)
	if err != nil {
		return err
	}
	// 记录历史
	if alarmRecord {
		alarmHistorys = append(alarmHistorys, *alarm)
	}
	return nil
}
