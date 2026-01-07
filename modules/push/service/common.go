package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// COMMON_PUSH_URI 通用推送URI地址 POST
const COMMON_PUSH_URI = "/push/common/receive"

// Common 通用服务
type Common struct {
	commonHistorys        sync.Map     // commonHistorys 通用历史记录
	commonHistorysMaxSize atomic.Int32 // 最大历史记录数
}

// NewCommon 创建通用服务
func NewCommon() *Common {
	c := &Common{}
	c.commonHistorysMaxSize.Store(4096)
	return c
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *Common) HistoryList(typeStr string, n int) []model.Common {
	if s == nil {
		return []model.Common{}
	}
	// 检查n是否小于0，如果是则返回空列表
	if n < 0 {
		return []model.Common{}
	}

	// 获取历史记录
	history, ok := s.commonHistorys.Load(typeStr)
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
func (s *Common) safeAppendCommonHistory(typeStr string, common *model.Common) {
	if s == nil {
		return
	}
	history, _ := s.commonHistorys.LoadOrStore(typeStr, []model.Common{})
	commonHistorysList := history.([]model.Common)

	maxSize := s.commonHistorysMaxSize.Load()
	newHistorys := make([]model.Common, len(commonHistorysList)+1)
	copy(newHistorys, commonHistorysList)
	newHistorys[len(newHistorys)-1] = *common

	if len(newHistorys) > int(maxSize) {
		newHistorys = newHistorys[len(newHistorys)-int(maxSize):]
	}

	s.commonHistorys.Store(typeStr, newHistorys)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *Common) HistorySetSize(newSize int) {
	if s == nil {
		return
	}
	oldSize := s.commonHistorysMaxSize.Swap(int32(newSize))
	if newSize < int(oldSize) {
		s.commonHistorys.Range(func(key, value interface{}) bool {
			if history, ok := value.([]model.Common); ok {
				if len(history) > newSize {
					s.commonHistorys.Store(key, history[len(history)-newSize:])
				}
			}
			return true
		})
	}
}

// PushURL 通用推送 自定义URL地址接收
func (s *Common) PushURL(url string, common *model.Common) error {
	if s == nil {
		return nil
	}
	common.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.safeAppendCommonHistory(common.Type, common)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	return fetch.AsyncPush(ctx, url, common)
}
