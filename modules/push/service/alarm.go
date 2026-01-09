package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/framework/utils"
	"github.com/tsmask/go-oam/modules/push/model"
)

// ALARM_PUSH_URI 告警推送URI地址 POST
const ALARM_PUSH_URI = "/push/alarm/receive"

// Alarm 告警服务
type Alarm struct {
	alarmHistorys        *utils.RingBuffer[model.Alarm] // 告警历史记录（环形缓冲区）
	alarmHistorysMaxSize atomic.Int32                   // 最大历史记录数量
}

// NewAlarm 创建告警服务
func NewAlarm() *Alarm {
	a := &Alarm{
		alarmHistorys: utils.NewRingBuffer[model.Alarm](4096),
	}
	a.alarmHistorysMaxSize.Store(4096)
	return a
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *Alarm) HistoryList(n int) []model.Alarm {
	if s == nil {
		return []model.Alarm{}
	}

	if n < 0 {
		return []model.Alarm{}
	}

	if n == 0 {
		return s.alarmHistorys.GetAll()
	}

	return s.alarmHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *Alarm) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	s.alarmHistorysMaxSize.Store(int32(newSize))
	s.alarmHistorys.Resize(newSize)
}

// PushURL 告警推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *Alarm) PushURL(url string, alarm *model.Alarm, timeout time.Duration) error {
	if s == nil {
		return nil
	}

	alarm.AlarmTime = time.Now().UnixMilli()

	// 记录历史
	s.alarmHistorys.Push(*alarm)

	// 发送
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, alarm)
}
