package push

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

const (
	NB_STATE_ON  = "ON"  // 基站状态-开
	NB_STATE_OFF = "OFF" // 基站状态-关
)

// NB_STATE_PUSH_URI 基站状态推送URI地址 POST
const NB_STATE_PUSH_URI = "/push/nb/state/receive"

// NBState 基站状态
type NBState struct {
	NeUid      string `json:"neUid" binding:"required"`       // 网元唯一标识
	RecordTime int64  `json:"recordTime" binding:"required"`  // 记录时间 时间戳毫秒，Push时自动填充
	Address    string `json:"address"  binding:"required"`    // 基站地址
	DeviceName string `json:"deviceName"  binding:"required"` // 基站设备名称
	DeviceId   int64  `json:"deviceId"  binding:"required"`   // 基站设备ID
	State      string `json:"state"  binding:"required"`      // 基站状态 ON/OFF
	StateTime  int64  `json:"stateTime"  binding:"required"`  // 基站状态时间 时间戳毫秒
	Name       string `json:"name"  binding:"required"`       // 基站名称 网元标记
	Position   string `json:"position"  binding:"required"`   // 基站位置 网元标记
}

// NBStateService 基站状态服务
type NBStateService struct {
	historys   *ringbuffer.RingBuffer[NBState]
	maxSize    atomic.Int32
}

// NewNBStateService 创建基站状态服务
func NewNBStateService() *NBStateService {
	n := &NBStateService{
		historys: ringbuffer.NewRingBuffer[NBState](4096),
	}
	n.maxSize.Store(4096)
	return n
}

// HistoryList 线程安全地获取历史列表
func (s *NBStateService) HistoryList(n int) []NBState {
	if s == nil {
		return []NBState{}
	}
	if n < 0 {
		return []NBState{}
	}
	if n == 0 {
		return s.historys.GetAll()
	}
	return s.historys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
func (s *NBStateService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.maxSize.Store(int32(newSize))
	s.historys.Resize(newSize)
}

// PushURL 基站状态推送
func (s *NBStateService) PushURL(url string, nbState *NBState, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	nbState.RecordTime = time.Now().UnixMilli()
	s.historys.Push(*nbState)
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, nbState)
}
