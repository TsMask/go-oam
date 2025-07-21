package service

import (
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// NB_STATE_PUSH_URI 基站状态推送URI地址 POST
const NB_STATE_PUSH_URI = "/push/nb/state/receive"

// nbStateRecord 控制是否记录历史基站状态
var nbStateRecord bool = false

// nbStateHistorys 基站状态历史 每小时重新记录，保留一小时
var nbStateHistorys []model.NBState = make([]model.NBState, 0)

// nbStateClearDuration 计算到下一个时间间隔
func nbStateClearDuration() time.Duration {
	now := time.Now()
	// 下一个整点
	nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	return nextHour.Sub(now)
}

// NBStateHistoryClearTimer 历史清除
func NBStateHistoryClearTimer() {
	// 启动时允许记录历史基站状态
	nbStateRecord = true
	// 创建一个定时器，在计算出的时间后触发第一次执行
	timer := time.NewTimer(nbStateClearDuration())

	go func() {
		for {
			now := <-timer.C
			// 计算小时前的起始时间戳
			oneHourAgo := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 0, 0, 0, now.Location()).UnixMilli()

			// 过滤出最近小时的 NBState 记录
			var recentHistory []model.NBState
			for _, NBState := range nbStateHistorys {
				if NBState.RecordTime >= oneHourAgo {
					recentHistory = append(recentHistory, NBState)
				}
			}
			// 更新 nbStateHistorys 为最近小时的记录
			nbStateHistorys = recentHistory

			// 重置历史记录清空定时器，准备下一个整点
			timer.Reset(nbStateClearDuration())
		}
	}()
}

// NBStateHistoryList 历史列表
func NBStateHistoryList() []model.NBState {
	return nbStateHistorys
}

// NBStatePushURL 基站状态推送 自定义URL地址接收
func NBStatePushURL(url string, nbState *model.NBState) error {
	nbState.RecordTime = time.Now().UnixMilli()

	// 记录历史
	if nbStateRecord {
		nbStateHistorys = append(nbStateHistorys, *nbState)
	}

	// 发送
	_, err := fetch.PostJSON(url, nbState, nil)
	if err != nil {
		return err
	}
	return nil
}
