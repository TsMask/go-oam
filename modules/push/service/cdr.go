package service

import (
	"context"
	"sync"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// CDR_PUSH_URI 话单推送URI地址 POST
const CDR_PUSH_URI = "/push/cdr/receive"

// CDR 话单服务
type CDR struct {
	cdrHistorys           []model.CDR  // 话单历史记录
	cdrHistorysMux        sync.RWMutex // 保护cdrHistorys的并发访问
	cdrHistorysMaxSize    int          // 最大历史记录数量
	cdrHistorysMaxSizeMux sync.RWMutex // 保护修改数量的并发访问
}

// NewCDR 创建话单服务
func NewCDR() *CDR {
	return &CDR{
		cdrHistorys:        []model.CDR{},
		cdrHistorysMaxSize: 4096,
	}
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *CDR) HistoryList(n int) []model.CDR {
	if s == nil {
		return []model.CDR{}
	}
	s.cdrHistorysMux.RLock()
	defer s.cdrHistorysMux.RUnlock()

	if n < 0 {
		return []model.CDR{}
	}

	// 计算要返回的记录数量
	historyLen := len(s.cdrHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.CDR, historyLen-startIndex)
	copy(result, s.cdrHistorys[startIndex:])
	return result
}

// safeAppendHistory 线程安全地添加话单历史记录
func (s *CDR) safeAppendHistory(cdr model.CDR) {
	if s == nil {
		return
	}
	s.cdrHistorysMux.Lock()
	defer s.cdrHistorysMux.Unlock()

	// 获取最大历史记录数
	s.cdrHistorysMaxSizeMux.RLock()
	maxSize := s.cdrHistorysMaxSize
	s.cdrHistorysMaxSizeMux.RUnlock()

	if len(s.cdrHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		s.cdrHistorys = s.cdrHistorys[1:]
	}

	s.cdrHistorys = append(s.cdrHistorys, cdr)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *CDR) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	s.cdrHistorysMaxSizeMux.Lock()
	oldSize := s.cdrHistorysMaxSize
	s.cdrHistorysMaxSize = newSize
	s.cdrHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		s.cdrHistorysMux.Lock()
		defer s.cdrHistorysMux.Unlock()

		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(s.cdrHistorys) > s.cdrHistorysMaxSize {
			s.cdrHistorys = s.cdrHistorys[len(s.cdrHistorys)-s.cdrHistorysMaxSize:]
		}
	}
}

// PushURL 话单推送 自定义URL地址接收
func (s *CDR) PushURL(url string, cdr *model.CDR) error {
	cdr.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.safeAppendHistory(*cdr)

	// 发送
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return fetch.Push(ctx, url, cdr)
}
