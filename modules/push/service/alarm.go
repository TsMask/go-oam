package service

import (
	"context"
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// ALARM_PUSH_URI 告警推送URI地址 POST
const ALARM_PUSH_URI = "/push/alarm/receive"

// Alarm 告警服务
type Alarm struct {
	alarmHistorys           []model.Alarm // 告警历史记录
	alarmHistorysMux        sync.RWMutex  // 保护alarmHistorys的并发访问
	alarmHistorysMaxSize    int           // 最大历史记录数量
	alarmHistorysMaxSizeMux sync.RWMutex  // 保护修改数量的并发访问
}

// NewAlarm 创建告警服务
func NewAlarm() *Alarm {
	return &Alarm{
		alarmHistorys:        []model.Alarm{},
		alarmHistorysMaxSize: 4096,
	}
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

	// 获取最大历史记录数
	s.alarmHistorysMaxSizeMux.RLock()
	maxSize := s.alarmHistorysMaxSize
	s.alarmHistorysMaxSizeMux.RUnlock()

	if len(s.alarmHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		s.alarmHistorys = s.alarmHistorys[1:]
	}

	s.alarmHistorys = append(s.alarmHistorys, alarm)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *Alarm) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	s.alarmHistorysMaxSizeMux.Lock()
	oldSize := s.alarmHistorysMaxSize
	s.alarmHistorysMaxSize = newSize
	s.alarmHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		s.alarmHistorysMux.Lock()
		defer s.alarmHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(s.alarmHistorys) > s.alarmHistorysMaxSize {
			s.alarmHistorys = s.alarmHistorys[len(s.alarmHistorys)-s.alarmHistorysMaxSize:]
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
	return fetch.Push(ctx, url, alarm)
}
