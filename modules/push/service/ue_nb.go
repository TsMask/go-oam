package service

import (
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// UENB_PUSH_URI 终端接入基站推送URI地址 POST
const UENB_PUSH_URI = "/push/ue/nb/receive"

var (
	uenbHistorys           []model.UENB // 终端接入基站历史记录
	uenbHistorysMux        sync.RWMutex // 保护uenbHistorys的并发访问
	uenbHistorysMaxSize    = 4096       // 最大历史记录数量
	uenbHistorysMaxSizeMux sync.RWMutex // 保护修改数量的并发访问
)

// UENBHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func UENBHistoryList(n int) []model.UENB {
	uenbHistorysMux.RLock()
	defer uenbHistorysMux.RUnlock()

	if n < 0 {
		return []model.UENB{}
	}

	// 计算要返回的记录数量
	historyLen := len(kpiHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.UENB, historyLen-startIndex)
	copy(result, uenbHistorys[startIndex:])
	return result
}

// safeAppendUENBHistory 线程安全地添加终端接入基站历史记录
func safeAppendUENBHistory(uenb model.UENB) {
	// 再获取写锁修改数据
	uenbHistorysMux.Lock()
	defer uenbHistorysMux.Unlock()

	// 获取最大历史记录数
	uenbHistorysMaxSizeMux.RLock()
	maxSize := uenbHistorysMaxSize
	uenbHistorysMaxSizeMux.RUnlock()

	if len(uenbHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		uenbHistorys = uenbHistorys[1:]
	}

	uenbHistorys = append(uenbHistorys, uenb)
}

// UENBHistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func UENBHistorySetSize(newSize int) {
	if newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	uenbHistorysMaxSizeMux.Lock()
	oldSize := uenbHistorysMaxSize
	uenbHistorysMaxSize = newSize
	uenbHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		uenbHistorysMux.Lock()
		defer uenbHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(uenbHistorys) > uenbHistorysMaxSize {
			uenbHistorys = uenbHistorys[len(uenbHistorys)-uenbHistorysMaxSize:]
		}
	}
}

// UENBPushURL 终端接入基站推送 自定义URL地址接收
func UENBPushURL(url string, uenb *model.UENB) error {
	uenb.RecordTime = time.Now().UnixMilli()

	// 记录历史
	safeAppendUENBHistory(*uenb)

	// 发送
	_, err := fetch.PostJSON(url, uenb, nil)
	if err != nil {
		return err
	}
	return nil
}
