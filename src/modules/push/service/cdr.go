package service

import (
	"time"

	"github.com/tsmask/go-oam/src/framework/fetch"
	"github.com/tsmask/go-oam/src/modules/push/model"
)

// CDR_PUSH_URI 话单推送URI地址 POST
const CDR_PUSH_URI = "/push/cdr/receive"

// cdrRecord 控制是否记录历史话单
var cdrRecord bool = false

// cdrHistorys 话单历史 每十分钟重新记录，保留十分钟
var cdrHistorys []model.CDR = make([]model.CDR, 0)

// cdrClearDuration 计算到下一个时间间隔
func cdrClearDuration() time.Duration {
	now := time.Now()
	// 下一个十分钟
	nextMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+10, 0, 0, now.Location())
	return nextMinute.Sub(now)
}

// CDRHistoryClearTimer 历史清除
func CDRHistoryClearTimer() {
	// 启动时允许记录历史话单
	cdrRecord = true
	// 创建一个定时器，在计算出的时间后触发第一次执行
	timer := time.NewTimer(cdrClearDuration())

	go func() {
		for {
			now := <-timer.C
			// 计算分钟前的起始时间戳
			minuteAgo := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-10, 0, 0, now.Location()).UnixMilli()

			// 过滤出最近分钟的 CDR 记录
			var recentHistory []model.CDR
			for _, cdr := range cdrHistorys {
				if cdr.RecordTime >= minuteAgo {
					recentHistory = append(recentHistory, cdr)
				}
			}
			// 更新 cdrHistorys 为最近分钟的记录
			cdrHistorys = recentHistory

			// 重置历史记录清空定时器，准备下一个十分钟
			timer.Reset(cdrClearDuration())
		}
	}()
}

// CDRHistoryList 历史列表
func CDRHistoryList() []model.CDR {
	return cdrHistorys
}

// CDRPushURL 话单推送 自定义URL地址接收
func CDRPushURL(url string, cdr *model.CDR) error {
	cdr.RecordTime = time.Now().UnixMilli()

	// 记录历史
	if cdrRecord {
		cdrHistorys = append(cdrHistorys, *cdr)
	}

	// 发送
	_, err := fetch.PostJSON(url, cdr, nil)
	if err != nil {
		return err
	}
	return nil
}
