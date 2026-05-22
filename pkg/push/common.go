package push

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

// COMMON_PUSH_URI 通用推送URI地址 POST
const COMMON_PUSH_URI = "/push/common/receive"

// Common 通用信息对象
type Common struct {
	NeUid      string `json:"neUid" binding:"required"`
	RecordTime int64  `json:"recordTime" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Data       any    `json:"data"  binding:"required"`
}

// CommonService 通用服务
type CommonService struct {
	historysMap sync.Map
	maxSize     atomic.Int32
}

// NewCommonService 创建通用服务
func NewCommonService() *CommonService {
	c := &CommonService{}
	c.maxSize.Store(4096)
	return c
}

func (s *CommonService) getOrCreateRingBuffer(typeStr string) *ringbuffer.RingBuffer[Common] {
	if s == nil {
		return nil
	}
	if val, ok := s.historysMap.Load(typeStr); ok {
		return val.(*ringbuffer.RingBuffer[Common])
	}
	mx := s.maxSize.Load()
	newBuffer := ringbuffer.NewRingBuffer[Common](int(mx))
	actual, _ := s.historysMap.LoadOrStore(typeStr, newBuffer)
	return actual.(*ringbuffer.RingBuffer[Common])
}

// HistoryList 线程安全地获取历史列表
func (s *CommonService) HistoryList(typeStr string, n int) []Common {
	if s == nil {
		return []Common{}
	}
	if n < 0 {
		return []Common{}
	}
	rb := s.getOrCreateRingBuffer(typeStr)
	if rb == nil {
		return []Common{}
	}
	if n == 0 {
		return rb.GetAll()
	}
	return rb.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
func (s *CommonService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.maxSize.Store(int32(newSize))
	s.historysMap.Range(func(_, value any) bool {
		value.(*ringbuffer.RingBuffer[Common]).Resize(newSize)
		return true
	})
}

// PushURL 通用推送
func (s *CommonService) PushURL(url string, common *Common, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	common.RecordTime = time.Now().UnixMilli()
	rb := s.getOrCreateRingBuffer(common.Type)
	if rb != nil {
		rb.Push(*common)
	}
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, common)
}
