package service

import (
	"context"
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// NB_STATE_PUSH_URI 基站状态推送URI地址 POST
const NB_STATE_PUSH_URI = "/push/nb/state/receive"

var (
	nbStateHistorys           []model.NBState // NB状态历史记录
	nbStateHistorysMux        sync.RWMutex    // 保护nbStateHistorys的并发访问
	nbStateHistorysMaxSize    = 4096          // 最大历史记录数量
	nbStateHistorysMaxSizeMux sync.RWMutex    // 保护修改数量的并发访问
)

// NBStateHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func NBStateHistoryList(n int) []model.NBState {
	nbStateHistorysMux.RLock()
	defer nbStateHistorysMux.RUnlock()

	if n < 0 {
		return []model.NBState{}
	}

	// 计算要返回的记录数量
	historyLen := len(nbStateHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.NBState, historyLen-startIndex)
	copy(result, nbStateHistorys[startIndex:])
	return result
}

// safeAppendNBStateHistory 线程安全地添加NB状态历史记录
func safeAppendNBStateHistory(state model.NBState) {
	// 再获取写锁修改数据
	nbStateHistorysMux.Lock()
	defer nbStateHistorysMux.Unlock()

	// 获取最大历史记录数
	nbStateHistorysMaxSizeMux.RLock()
	maxSize := nbStateHistorysMaxSize
	nbStateHistorysMaxSizeMux.RUnlock()

	if len(nbStateHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		nbStateHistorys = nbStateHistorys[1:]
	}

	nbStateHistorys = append(nbStateHistorys, state)
}

// NBStateHistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func NBStateHistorySetSize(newSize int) {
	if newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	nbStateHistorysMaxSizeMux.Lock()
	oldSize := nbStateHistorysMaxSize
	nbStateHistorysMaxSize = newSize
	nbStateHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		nbStateHistorysMux.Lock()
		defer nbStateHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(nbStateHistorys) > nbStateHistorysMaxSize {
			nbStateHistorys = nbStateHistorys[len(nbStateHistorys)-nbStateHistorysMaxSize:]
		}
	}
}

// NBStatePushURL 基站状态推送 自定义URL地址接收
func NBStatePushURL(url string, nbState *model.NBState) error {
	nbState.RecordTime = time.Now().UnixMilli()

	// 记录历史
	safeAppendNBStateHistory(*nbState)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return fetch.Push(ctx, url, nbState)
}
