package service

import (
	"sync"
	"time"

	"github.com/tsmask/go-oam/src/framework/fetch"
	"github.com/tsmask/go-oam/src/modules/push/model"
)

// COMMON_PUSH_URI 通用推送URI地址 POST
const COMMON_PUSH_URI = "/push/common/receive"

// commonRecord 控制是否记录历史通用信息
var commonRecord bool = false

// commonHistorys 通用历史记录
var commonHistorys sync.Map

// CommonHistoryClearTimer 历史清除定时器
func CommonHistoryClearTimer(typeStr string, d time.Duration) {
	// 启动时允许记录历史通用信息
	commonRecord = true
	// 创建一个定时器，在计算出的时间后触发第一次执行
	timer := time.NewTimer(d)

	go func() {
		for {
			<-timer.C
			// 清除历史数据
			commonHistorys.Store(typeStr, make([]model.Common, 0))
			// 重置历史记录清空定时器，准备下一个时间间隔
			timer.Reset(d)
		}
	}()
}

// CommonHistoryList 历史列表
func CommonHistoryList(typeStr string) []model.Common {
	history, ok := commonHistorys.Load(typeStr)
	if !ok {
		return []model.Common{}
	}
	return history.([]model.Common)
}

// CommonPushURL 通用推送 自定义URL地址接收
func CommonPushURL(url string, common *model.Common) error {
	common.RecordTime = time.Now().UnixMilli()

	// 记录历史
	if commonRecord {
		history, ok := commonHistorys.Load(common.Type)
		if !ok {
			history = make([]model.Common, 0)
		}
		commonHistorys.Store(common.Type, append(history.([]model.Common), *common))
	}

	// 发送
	_, err := fetch.PostJSON(url, common, nil)
	if err != nil {
		return err
	}
	return nil
}
