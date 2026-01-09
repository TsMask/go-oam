package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/framework/utils"
	"github.com/tsmask/go-oam/modules/push/model"
)

// COMMON_PUSH_URI 通用推送URI地址 POST
const COMMON_PUSH_URI = "/push/common/receive"

// Common 通用服务
type Common struct {
	commonHistorysMap     sync.Map     //  通用历史记录
	commonHistorysMaxSize atomic.Int32 // 最大历史记录数
}

// NewCommon 创建通用服务
func NewCommon() *Common {
	c := &Common{}
	c.commonHistorysMaxSize.Store(4096)
	return c
}

// getOrCreateRingBuffer 获取或创建指定类型的环形缓冲区
func (s *Common) getOrCreateRingBuffer(typeStr string) *utils.RingBuffer[model.Common] {
	if s == nil {
		return nil
	}

	if val, ok := s.commonHistorysMap.Load(typeStr); ok {
		return val.(*utils.RingBuffer[model.Common])
	}

	maxSize := s.commonHistorysMaxSize.Load()
	newBuffer := utils.NewRingBuffer[model.Common](int(maxSize))
	actual, _ := s.commonHistorysMap.LoadOrStore(typeStr, newBuffer)
	return actual.(*utils.RingBuffer[model.Common])
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

	rb := s.getOrCreateRingBuffer(typeStr)
	if rb == nil {
		return []model.Common{}
	}

	if n == 0 {
		return rb.GetAll()
	}

	return rb.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *Common) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	s.commonHistorysMaxSize.Store(int32(newSize))

	s.commonHistorysMap.Range(func(key, value any) bool {
		rb := value.(*utils.RingBuffer[model.Common])
		rb.Resize(newSize)
		return true
	})
}

// PushURL 通用推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *Common) PushURL(url string, common *model.Common, timeout time.Duration) error {
	if s == nil {
		return nil
	}

	common.RecordTime = time.Now().UnixMilli()

	// 记录历史
	rb := s.getOrCreateRingBuffer(common.Type)
	if rb != nil {
		rb.Push(*common)
	}

	// 发送
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, common)
}
