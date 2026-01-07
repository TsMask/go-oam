package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// ALARM_PUSH_URI 告警推送URI地址 POST
const ALARM_PUSH_URI = "/push/alarm/receive"

// Alarm 告警服务
type Alarm struct {
	alarmHistorys        []model.Alarm // 告警历史记录
	alarmHistorysMux     sync.RWMutex  // 保护alarmHistorys的并发访问
	alarmHistorysMaxSize atomic.Int32  // 最大历史记录数量
}

// NewAlarm 创建告警服务
func NewAlarm() *Alarm {
	a := &Alarm{
		alarmHistorys: []model.Alarm{},
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
	s.alarmHistorysMux.RLock()
	defer s.alarmHistorysMux.RUnlock()

	if n < 0 {
		return []model.Alarm{}
	}

	// 计算要返回的记录数量
	historyLen := len(s.alarmHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.Alarm, historyLen-startIndex)
	copy(result, s.alarmHistorys[startIndex:])
	return result
}

// safeAppendHistory 线程安全地添加历史记录
func (s *Alarm) safeAppendHistory(alarm model.Alarm) {
	if s == nil {
		return
	}
	s.alarmHistorysMux.Lock()
	defer s.alarmHistorysMux.Unlock()

	maxSize := s.alarmHistorysMaxSize.Load()
	if len(s.alarmHistorys) >= int(maxSize) {
		s.alarmHistorys = s.alarmHistorys[1:]
	}
	s.alarmHistorys = append(s.alarmHistorys, alarm)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *Alarm) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	oldSize := s.alarmHistorysMaxSize.Swap(int32(newSize))
	if newSize < int(oldSize) {
		s.alarmHistorysMux.Lock()
		defer s.alarmHistorysMux.Unlock()

		if len(s.alarmHistorys) > newSize {
			s.alarmHistorys = s.alarmHistorys[len(s.alarmHistorys)-newSize:]
		}
	}
}

// PushURL 告警推送 自定义URL地址接收
func (s *Alarm) PushURL(url string, alarm *model.Alarm) error {
	alarm.AlarmTime = time.Now().UnixMilli()

	// 记录历史
	s.safeAppendHistory(*alarm)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	return fetch.AsyncPush(ctx, url, alarm)
}
