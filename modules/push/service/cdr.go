package service

import (
	"context"
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// CDR_PUSH_URI 话单推送URI地址 POST
const CDR_PUSH_URI = "/push/cdr/receive"

var (
	cdrHistorys           []model.CDR  // 话单历史记录
	cdrHistorysMux        sync.RWMutex // 保护cdrHistorys的并发访问
	cdrHistorysMaxSize    = 4096       // 最大历史记录数量
	cdrHistorysMaxSizeMux sync.RWMutex // 保护修改数量的并发访问
)

// CDRHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func CDRHistoryList(n int) []model.CDR {
	cdrHistorysMux.RLock()
	defer cdrHistorysMux.RUnlock()

	if n < 0 {
		return []model.CDR{}
	}

	// 计算要返回的记录数量
	historyLen := len(cdrHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.CDR, historyLen-startIndex)
	copy(result, cdrHistorys[startIndex:])
	return result
}

// safeAppendCDRHistory 线程安全地添加话单历史记录
func safeAppendCDRHistory(cdr model.CDR) {
	cdrHistorysMux.Lock()
	defer cdrHistorysMux.Unlock()

	// 获取最大历史记录数
	cdrHistorysMaxSizeMux.RLock()
	maxSize := cdrHistorysMaxSize
	cdrHistorysMaxSizeMux.RUnlock()

	if len(cdrHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		cdrHistorys = cdrHistorys[1:]
	}

	cdrHistorys = append(cdrHistorys, cdr)
}

// CDRHistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func CDRHistorySetSize(newSize int) {
	if newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	cdrHistorysMaxSizeMux.Lock()
	oldSize := cdrHistorysMaxSize
	cdrHistorysMaxSize = newSize
	cdrHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		cdrHistorysMux.Lock()
		defer cdrHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(cdrHistorys) > cdrHistorysMaxSize {
			cdrHistorys = cdrHistorys[len(cdrHistorys)-cdrHistorysMaxSize:]
		}
	}
}

// CDRPushURL 话单推送 自定义URL地址接收
func CDRPushURL(url string, cdr *model.CDR) error {
	cdr.RecordTime = time.Now().UnixMilli()

	// 记录历史
	safeAppendCDRHistory(*cdr)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := fetch.EnqueuePush(url, cdr); err != nil {
		_, err := fetch.PostJSON(ctx, url, cdr, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
