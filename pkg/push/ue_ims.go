package push

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

const (
	UEIMS_RESULT_UNKNOWN = "0"
	UEIMS_RESULT_SUCCESS = "200"
)

const (
	UEIMS_TYPE_REGISTER   = "InitialRegister"
	UEIMS_TYPE_PERIODIC   = "PeriodicRegister"
	UEIMS_TYPE_UNREGISTER = "Unregister"
)

// UEIMS_PUSH_URI 终端接入IMS推送URI地址 POST
const UEIMS_PUSH_URI = "/push/ue/ims/receive"

// UEIMS 终端接入IMS信息对象
type UEIMS struct {
	NeUid      string `json:"neUid" binding:"required"`
	RecordTime int64  `json:"recordTime" binding:"required"`
	IMSI       string `json:"imsi"  binding:"required"`
	Result     string `json:"result" binding:"required"`
	Type       string `json:"type" binding:"required"`
}

// UEMISService 终端接入IMS服务
type UEMISService struct {
	historys *ringbuffer.RingBuffer[UEIMS]
	maxSize  atomic.Int32
}

// NewUEMISService 创建终端接入IMS服务
func NewUEMISService() *UEMISService {
	u := &UEMISService{
		historys: ringbuffer.NewRingBuffer[UEIMS](4096),
	}
	u.maxSize.Store(4096)
	return u
}

func (s *UEMISService) HistoryList(n int) []UEIMS {
	if s == nil {
		return []UEIMS{}
	}
	if n < 0 {
		return []UEIMS{}
	}
	if n == 0 {
		return s.historys.GetAll()
	}
	return s.historys.GetLast(n)
}

func (s *UEMISService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.maxSize.Store(int32(newSize))
	s.historys.Resize(newSize)
}

func (s *UEMISService) PushURL(url string, ueims *UEIMS, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	ueims.RecordTime = time.Now().UnixMilli()
	s.historys.Push(*ueims)
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, ueims)
}
