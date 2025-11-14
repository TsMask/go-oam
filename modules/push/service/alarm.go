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

var (
	alarmHistorys           []model.Alarm // 告警历史记录
	alarmHistorysMux        sync.RWMutex  // 保护alarmHistorys的并发访问
	alarmHistorysMaxSize    = 4096        // 最大历史记录数量
	alarmHistorysMaxSizeMux sync.RWMutex  // 保护修改数量的并发访问
)

// AlarmHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func AlarmHistoryList(n int) []model.Alarm {
	alarmHistorysMux.RLock()
	defer alarmHistorysMux.RUnlock()

	if n < 0 {
		return []model.Alarm{}
	}

	// 计算要返回的记录数量
	historyLen := len(alarmHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.Alarm, historyLen-startIndex)
	copy(result, alarmHistorys[startIndex:])
	return result
}

// safeAppendAlarmHistory 线程安全地添加告警历史记录
func safeAppendAlarmHistory(alarm model.Alarm) {
	alarmHistorysMux.Lock()
	defer alarmHistorysMux.Unlock()

	// 获取最大历史记录数
	alarmHistorysMaxSizeMux.RLock()
	maxSize := alarmHistorysMaxSize
	alarmHistorysMaxSizeMux.RUnlock()

	if len(alarmHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		alarmHistorys = alarmHistorys[1:]
	}

	alarmHistorys = append(alarmHistorys, alarm)
}

// AlarmHistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func AlarmHistorySetSize(newSize int) {
	if newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	alarmHistorysMaxSizeMux.Lock()
	oldSize := alarmHistorysMaxSize
	alarmHistorysMaxSize = newSize
	alarmHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		alarmHistorysMux.Lock()
		defer alarmHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(alarmHistorys) > alarmHistorysMaxSize {
			alarmHistorys = alarmHistorys[len(alarmHistorys)-alarmHistorysMaxSize:]
		}
	}
}

// AlarmPushURL 告警推送 自定义URL地址接收
func AlarmPushURL(url string, alarm *model.Alarm) error {
	alarm.AlarmTime = time.Now().UnixMilli()

	// 记录历史
	safeAppendAlarmHistory(*alarm)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := fetch.EnqueuePush(url, alarm); err != nil {
		_, err := fetch.PostJSON(ctx, url, alarm, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
