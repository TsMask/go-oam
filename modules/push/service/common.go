package service

import (
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// COMMON_PUSH_URI 通用推送URI地址 POST
const COMMON_PUSH_URI = "/push/common/receive"

var (
	commonHistorys           sync.Map     // commonHistorys 通用历史记录
	commonHistorysMaxSizeMux sync.RWMutex // 保护最大历史记录数的锁
	commonHistorysMaxSize    = 4096       // 最大历史记录数
)

// CommonHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func CommonHistoryList(typeStr string, n int) []model.Common {
	// 检查n是否小于0，如果是则返回空列表
	if n < 0 {
		return []model.Common{}
	}

	// 获取历史记录
	history, ok := commonHistorys.Load(typeStr)
	if !ok {
		return []model.Common{}
	}

	// 类型断言
	commonHistorysList, ok := history.([]model.Common)
	if !ok {
		return []model.Common{}
	}

	// 计算要返回的记录起始索引
	historyLen := len(commonHistorysList)
	startIndex := 0
	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分，避免不必要的内存分配
	result := make([]model.Common, historyLen-startIndex)
	copy(result, commonHistorysList[startIndex:])
	return result
}

// safeAppendCommonHistory 线程安全地添加历史记录
func safeAppendCommonHistory(typeStr string, common *model.Common) {
	// 获取当前历史记录，如果不存在则创建空切片
	history, _ := commonHistorys.LoadOrStore(typeStr, []model.Common{})
	commonHistorysList := history.([]model.Common)

	// 获取最大历史记录数
	commonHistorysMaxSizeMux.RLock()
	maxSize := commonHistorysMaxSize
	commonHistorysMaxSizeMux.RUnlock()

	// 创建新的切片，避免直接修改原切片
	newHistorys := make([]model.Common, len(commonHistorysList)+1)
	copy(newHistorys, commonHistorysList)
	newHistorys[len(newHistorys)-1] = *common

	// 如果超过最大记录数，删除最旧的记录
	if len(newHistorys) > maxSize {
		newHistorys = newHistorys[len(newHistorys)-maxSize:]
	}

	// 存储更新后的历史记录
	commonHistorys.Store(typeStr, newHistorys)
}

// CommonHistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func CommonHistorySetSize(newSize int) {
	commonHistorysMaxSizeMux.Lock()
	oldSize := commonHistorysMaxSize
	commonHistorysMaxSize = newSize
	commonHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，需要清理历史记录
	if newSize < oldSize {
		commonHistorys.Range(func(key, value interface{}) bool {
			if history, ok := value.([]model.Common); ok {
				if len(history) > newSize {
					// 只保留最新的记录
					commonHistorys.Store(key, history[len(history)-newSize:])
				}
			}
			return true
		})
	}
}

// CommonPushURL 通用推送 自定义URL地址接收
func CommonPushURL(url string, common *model.Common) error {
	common.RecordTime = time.Now().UnixMilli()

	// 线程安全地记录历史
	safeAppendCommonHistory(common.Type, common)

	// 发送推送请求
	_, err := fetch.PostJSON(url, common, nil)
	if err != nil {
		return err
	}
	return nil
}
