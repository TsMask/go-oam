package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/framework/utils"
	"github.com/tsmask/go-oam/modules/push/model"
)

// UENB_PUSH_URI 终端接入基站推送URI地址 POST
const UENB_PUSH_URI = "/push/ue/nb/receive"

// UENB 终端接入基站服务
type UENB struct {
	uenbHistorys        *utils.RingBuffer[model.UENB] // 终端接入基站历史记录（环形缓冲区）
	uenbHistorysMaxSize atomic.Int32                  // 最大历史记录数量
}

// NewUENB 创建终端接入基站服务
func NewUENB() *UENB {
	u := &UENB{
		uenbHistorys: utils.NewRingBuffer[model.UENB](4096),
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

	if n < 0 {
		return []model.UENB{}
	}

	if n == 0 {
		return s.uenbHistorys.GetAll()
	}

	return s.uenbHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *UENB) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	s.uenbHistorysMaxSize.Store(int32(newSize))
	s.uenbHistorys.Resize(newSize)
}

// PushURL 终端接入基站推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *UENB) PushURL(url string, uenb *model.UENB, timeout time.Duration) error {
	if s == nil {
		return nil
	}

	uenb.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.uenbHistorys.Push(*uenb)

	// 发送
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, uenb)
}
