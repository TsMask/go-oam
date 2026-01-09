package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/framework/utils"
	"github.com/tsmask/go-oam/modules/push/model"
)

// CDR_PUSH_URI 话单推送URI地址 POST
const CDR_PUSH_URI = "/push/cdr/receive"

// CDR 话单服务
type CDR struct {
	cdrHistorys        *utils.RingBuffer[model.CDR] // 话单历史记录（环形缓冲区）
	cdrHistorysMaxSize atomic.Int32                 // 最大历史记录数量
}

// NewCDR 创建话单服务
func NewCDR() *CDR {
	c := &CDR{
		cdrHistorys: utils.NewRingBuffer[model.CDR](4096),
	}
	c.cdrHistorysMaxSize.Store(4096)
	return c
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *CDR) HistoryList(n int) []model.CDR {
	if s == nil {
		return []model.CDR{}
	}

	if n < 0 {
		return []model.CDR{}
	}

	if n == 0 {
		return s.cdrHistorys.GetAll()
	}

	return s.cdrHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *CDR) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	s.cdrHistorysMaxSize.Store(int32(newSize))
	s.cdrHistorys.Resize(newSize)
}

// PushURL 话单推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *CDR) PushURL(url string, cdr *model.CDR, timeout time.Duration) error {
	if s == nil {
		return nil
	}

	cdr.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.cdrHistorys.Push(*cdr)

	// 发送
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, cdr)
}
