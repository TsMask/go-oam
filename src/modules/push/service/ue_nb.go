package service

import (
	"time"

	"github.com/tsmask/go-oam/src/framework/fetch"
	"github.com/tsmask/go-oam/src/modules/push/model"
)

// UENB_PUSH_URI 终端接入基站推送URI地址 POST
const UENB_PUSH_URI = "/push/ue/nb/receive"

// uenbRecord 控制是否记录历史终端接入基站
var uenbRecord bool = false

// uenbHistorys 终端接入基站历史 每小时重新记录，保留一小时
var uenbHistorys []model.UENB = make([]model.UENB, 0)

// uenbClearDuration 计算到下一个时间间隔
func uenbClearDuration() time.Duration {
	now := time.Now()
	// 下一个整点
	nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	return nextHour.Sub(now)
}

// UENBHistoryClearTimer 历史清除
func UENBHistoryClearTimer() {
	// 启动时允许记录历史终端接入基站
	uenbRecord = true
	// 创建一个定时器，在计算出的时间后触发第一次执行
	timer := time.NewTimer(uenbClearDuration())

	go func() {
		for {
			now := <-timer.C
			// 计算小时前的起始时间戳
			oneHourAgo := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 0, 0, 0, now.Location()).UnixMilli()

			// 过滤出最近小时的 UENB 记录
			var recentHistory []model.UENB
			for _, uenb := range uenbHistorys {
				if uenb.RecordTime >= oneHourAgo {
					recentHistory = append(recentHistory, uenb)
				}
			}
			// 更新 uenbHistorys 为最近小时的记录
			uenbHistorys = recentHistory

			// 重置历史记录清空定时器，准备下一个整点
			timer.Reset(uenbClearDuration())
		}
	}()
}

// UENBHistoryList 历史列表
func UENBHistoryList() []model.UENB {
	return uenbHistorys
}

// UENBPushURL 终端接入基站推送 自定义URL地址接收
func UENBPushURL(url string, uenb *model.UENB) error {
	uenb.RecordTime = time.Now().UnixMilli()
	// 发送
	_, err := fetch.PostJSON(url, uenb, nil)
	if err != nil {
		return err
	}
	// 记录历史
	if uenbRecord {
		uenbHistorys = append(uenbHistorys, *uenb)
	}
	return nil
}
