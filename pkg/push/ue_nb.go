package push

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

const (
	UENB_RESULT_AUTH_SUCCESS                            = "200"
	UENB_RESULT_AUTH_NETWORK_FAILURE                    = "001"
	UENB_RESULT_AUTH_INTERFACE_FAILURE                  = "002"
	UENB_RESULT_AUTH_MAC_FAILURE                        = "003"
	UENB_RESULT_AUTH_SYNC_FAILURE                       = "004"
	UENB_RESULT_AUTH_NON_5G_AUTHENTICATION_NOT_ACCEPTED = "005"
	UENB_RESULT_AUTH_RESPONSE_FAILURE                   = "006"
	UENB_RESULT_AUTH_UNKNOWN                            = "007"
	UENB_RESULT_AUTH_ILLEGAL                            = "008"
	UENB_RESULT_CM_CONNECTED                            = "1"
	UENB_RESULT_CM_IDLE                                 = "2"
	UENB_RESULT_CM_INACTIVE                             = "3"
)

const (
	UENB_TYPE_AUTH     = "Auth"
	UENB_TYPE_DETACH   = "Detach"
	UENB_TYPE_CM       = "CM"
	UENB_TYPE_ATTACH   = "ATTACH"
	UENB_TYPE_TAU      = "TAU"
	UENB_TYPE_HANDOVER = "HANDOVER"
)

// UENB_PUSH_URI 终端接入基站推送URI地址 POST
const UENB_PUSH_URI = "/push/ue/nb/receive"

// UENB 终端接入基站信息对象 AMF/MME
type UENB struct {
	NeUid      string `json:"neUid" binding:"required"`
	RecordTime int64  `json:"recordTime" binding:"required"`
	NBId       string `json:"nbId" binding:"required"`
	NBIp       string `json:"nbIp"`
	CellId     string `json:"cellId" binding:"required"`
	TAC        string `json:"tac" binding:"required"`
	IMSI       string `json:"imsi" binding:"required"`
	MSISDN     string `json:"msisdn"`
	IMEI       string `json:"imei"`
	Result     string `json:"result" binding:"required"`
	Type       string `json:"type" binding:"required"`
}

// UENBService 终端接入基站服务
type UENBService struct {
	historys *ringbuffer.RingBuffer[UENB]
	maxSize  atomic.Int32
}

// NewUENBService 创建终端接入基站服务
func NewUENBService() *UENBService {
	u := &UENBService{
		historys: ringbuffer.NewRingBuffer[UENB](4096),
	}
	u.maxSize.Store(4096)
	return u
}

func (s *UENBService) HistoryList(n int) []UENB {
	if s == nil {
		return []UENB{}
	}
	if n < 0 {
		return []UENB{}
	}
	if n == 0 {
		return s.historys.GetAll()
	}
	return s.historys.GetLast(n)
}

func (s *UENBService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.maxSize.Store(int32(newSize))
	s.historys.Resize(newSize)
}

func (s *UENBService) PushURL(url string, uenb *UENB, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	uenb.RecordTime = time.Now().UnixMilli()
	s.historys.Push(*uenb)
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, uenb)
}
