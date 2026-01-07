package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// UENB_PUSH_URI 终端接入基站推送URI地址 POST
const UENB_PUSH_URI = "/push/ue/nb/receive"

// UENB 终端接入基站服务
type UENB struct {
	uenbHistorys        []model.UENB // 终端接入基站历史记录
	uenbHistorysMux     sync.RWMutex // 保护uenbHistorys的并发访问
	uenbHistorysMaxSize atomic.Int32 // 最大历史记录数量
}

// NewUENB 创建终端接入基站服务
func NewUENB() *UENB {
	u := &UENB{
		uenbHistorys: []model.UENB{},
	}
	u.uenbHistorysMaxSize.Store(4096)
	return u
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *UENB) HistoryList(n int) []model.UENB {
	if s == nil {
		return []model.UENB{}
	}
	s.uenbHistorysMux.RLock()
	defer s.uenbHistorysMux.RUnlock()

	if n < 0 {
		return []model.UENB{}
	}

	// 计算要返回的记录数量
	historyLen := len(s.uenbHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.UENB, historyLen-startIndex)
	copy(result, s.uenbHistorys[startIndex:])
	return result
}

// safeAppendHistory 线程安全地添加终端接入基站历史记录
func (s *UENB) safeAppendHistory(uenb model.UENB) {
	if s == nil {
		return
	}
	s.uenbHistorysMux.Lock()
	defer s.uenbHistorysMux.Unlock()

	maxSize := s.uenbHistorysMaxSize.Load()
	if len(s.uenbHistorys) >= int(maxSize) {
		s.uenbHistorys = s.uenbHistorys[1:]
	}

	s.uenbHistorys = append(s.uenbHistorys, uenb)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *UENB) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	oldSize := s.uenbHistorysMaxSize.Swap(int32(newSize))
	if newSize < int(oldSize) {
		s.uenbHistorysMux.Lock()
		defer s.uenbHistorysMux.Unlock()

		if len(s.uenbHistorys) > newSize {
			s.uenbHistorys = s.uenbHistorys[len(s.uenbHistorys)-newSize:]
		}
	}
}

// PushURL 终端接入基站推送 自定义URL地址接收
func (s *UENB) PushURL(url string, uenb *model.UENB) error {
	uenb.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.safeAppendHistory(*uenb)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	return fetch.AsyncPush(ctx, url, uenb)
}
