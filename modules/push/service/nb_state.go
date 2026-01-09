package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/framework/utils"
	"github.com/tsmask/go-oam/modules/push/model"
)

// NB_STATE_PUSH_URI 基站状态推送URI地址 POST
const NB_STATE_PUSH_URI = "/push/nb/state/receive"

// NBState NB状态服务
type NBState struct {
	nbStateHistorys        *utils.RingBuffer[model.NBState] // NB状态历史记录（环形缓冲区）
	nbStateHistorysMaxSize atomic.Int32                     // 最大历史记录数量
}

// NewNBState 创建NB状态服务
func NewNBState() *NBState {
	n := &NBState{
		nbStateHistorys: utils.NewRingBuffer[model.NBState](4096),
	}
	n.nbStateHistorysMaxSize.Store(4096)
	return n
}

// NBStateHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *NBState) HistoryList(n int) []model.NBState {
	if s == nil {
		return []model.NBState{}
	}

	if n < 0 {
		return []model.NBState{}
	}

	if n == 0 {
		return s.nbStateHistorys.GetAll()
	}

	return s.nbStateHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *NBState) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	s.nbStateHistorysMaxSize.Store(int32(newSize))
	s.nbStateHistorys.Resize(newSize)
}

// PushURL NB状态推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *NBState) PushURL(url string, nbState *model.NBState, timeout time.Duration) error {
	if s == nil {
		return nil
	}

	nbState.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.nbStateHistorys.Push(*nbState)

	// 发送
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, nbState)
}
