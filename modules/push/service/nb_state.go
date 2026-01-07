package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// NB_STATE_PUSH_URI 基站状态推送URI地址 POST
const NB_STATE_PUSH_URI = "/push/nb/state/receive"

// NBState NB状态服务
type NBState struct {
	nbStateHistorys        []model.NBState // NB状态历史记录
	nbStateHistorysMux     sync.RWMutex    // 保护nbStateHistorys的并发访问
	nbStateHistorysMaxSize atomic.Int32    // 最大历史记录数量
}

// NewNBState 创建NB状态服务
func NewNBState() *NBState {
	n := &NBState{
		nbStateHistorys: []model.NBState{},
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
	s.nbStateHistorysMux.RLock()
	defer s.nbStateHistorysMux.RUnlock()

	if n < 0 {
		return []model.NBState{}
	}

	// 计算要返回的记录数量
	historyLen := len(s.nbStateHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.NBState, historyLen-startIndex)
	copy(result, s.nbStateHistorys[startIndex:])
	return result
}

// safeAppendHistory 线程安全地添加NB状态历史记录
func (s *NBState) safeAppendHistory(state model.NBState) {
	if s == nil {
		return
	}
	s.nbStateHistorysMux.Lock()
	defer s.nbStateHistorysMux.Unlock()

	maxSize := s.nbStateHistorysMaxSize.Load()
	if len(s.nbStateHistorys) >= int(maxSize) {
		s.nbStateHistorys = s.nbStateHistorys[1:]
	}
	s.nbStateHistorys = append(s.nbStateHistorys, state)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *NBState) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	oldSize := s.nbStateHistorysMaxSize.Swap(int32(newSize))
	if newSize < int(oldSize) {
		s.nbStateHistorysMux.Lock()
		defer s.nbStateHistorysMux.Unlock()

		if len(s.nbStateHistorys) > newSize {
			s.nbStateHistorys = s.nbStateHistorys[len(s.nbStateHistorys)-newSize:]
		}
	}
}

// PushURL NB状态推送 自定义URL地址接收
func (s *NBState) PushURL(url string, nbState *model.NBState) error {
	nbState.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.safeAppendHistory(*nbState)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	return fetch.AsyncPush(ctx, url, nbState)
}
