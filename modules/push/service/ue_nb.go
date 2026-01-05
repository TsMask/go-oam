package service

import (
	"context"
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// UENB_PUSH_URI 终端接入基站推送URI地址 POST
const UENB_PUSH_URI = "/push/ue/nb/receive"

// UENB 终端接入基站服务
type UENB struct {
	uenbHistorys           []model.UENB // 终端接入基站历史记录
	uenbHistorysMux        sync.RWMutex // 保护uenbHistorys的并发访问
	uenbHistorysMaxSize    int          // 最大历史记录数量
	uenbHistorysMaxSizeMux sync.RWMutex // 保护修改数量的并发访问
}

// NewUENB 创建终端接入基站服务
func NewUENB() *UENB {
	return &UENB{
		uenbHistorys:        []model.UENB{},
		uenbHistorysMaxSize: 4096,
	}
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
	// 再获取写锁修改数据
	s.uenbHistorysMux.Lock()
	defer s.uenbHistorysMux.Unlock()

	// 获取最大历史记录数
	s.uenbHistorysMaxSizeMux.RLock()
	maxSize := s.uenbHistorysMaxSize
	s.uenbHistorysMaxSizeMux.RUnlock()

	if len(s.uenbHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		s.uenbHistorys = s.uenbHistorys[1:]
	}

	s.uenbHistorys = append(s.uenbHistorys, uenb)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *UENB) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	s.uenbHistorysMaxSizeMux.Lock()
	oldSize := s.uenbHistorysMaxSize
	s.uenbHistorysMaxSize = newSize
	s.uenbHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		s.uenbHistorysMux.Lock()
		defer s.uenbHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(s.uenbHistorys) > s.uenbHistorysMaxSize {
			s.uenbHistorys = s.uenbHistorys[len(s.uenbHistorys)-s.uenbHistorysMaxSize:]
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
