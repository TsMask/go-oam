package push

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

// CDR_PUSH_URI 话单推送URI地址 POST
const CDR_PUSH_URI = "/push/cdr/receive"

// CDR 话单信息对象
type CDR struct {
	NeUid      string `json:"neUid" binding:"required"`
	RecordTime int64  `json:"recordTime" binding:"required"`
	Data       any    `json:"data"  binding:"required"`
}

// CDRService 话单服务
type CDRService struct {
	historys *ringbuffer.RingBuffer[CDR]
	maxSize  atomic.Int32
}

// NewCDRService 创建话单服务
func NewCDRService() *CDRService {
	c := &CDRService{
		historys: ringbuffer.NewRingBuffer[CDR](4096),
	}
	c.maxSize.Store(4096)
	return c
}

func (s *CDRService) HistoryList(n int) []CDR {
	if s == nil {
		return []CDR{}
	}
	if n < 0 {
		return []CDR{}
	}
	if n == 0 {
		return s.historys.GetAll()
	}
	return s.historys.GetLast(n)
}

func (s *CDRService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.maxSize.Store(int32(newSize))
	s.historys.Resize(newSize)
}

func (s *CDRService) PushURL(url string, cdr *CDR, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	cdr.RecordTime = time.Now().UnixMilli()
	s.historys.Push(*cdr)
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, cdr)
}
