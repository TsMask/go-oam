package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/framework/utils"
	"github.com/tsmask/go-oam/modules/push/model"
)

// UEIMS_PUSH_URI 终端接入IMS推送URI地址 POST
const UEIMS_PUSH_URI = "/push/ue/ims/receive"

// UEIMS 终端接入IMS服务
type UEIMS struct {
	ueimsHistorys        *utils.RingBuffer[model.UEIMS] // 终端接入IMS历史记录（环形缓冲区）
	ueimsHistorysMaxSize atomic.Int32                   // 最大历史记录数量
}

// NewUEIMS 创建终端接入IMS服务
func NewUEIMS() *UEIMS {
	u := &UEIMS{
		ueimsHistorys: utils.NewRingBuffer[model.UEIMS](4096),
	}
	u.ueimsHistorysMaxSize.Store(4096)
	return u
}

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *UEIMS) HistoryList(n int) []model.UEIMS {
	if s == nil {
		return []model.UEIMS{}
	}

	if n < 0 {
		return []model.UEIMS{}
	}

	if n == 0 {
		return s.ueimsHistorys.GetAll()
	}

	return s.ueimsHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *UEIMS) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	s.ueimsHistorysMaxSize.Store(int32(newSize))
	s.ueimsHistorys.Resize(newSize)
}

// PushURL 终端接入IMS推送 自定义URL地址接收
// timeout: 超时时间，0 或负数表示使用默认值 1 分钟
func (s *UEIMS) PushURL(url string, ueims *model.UEIMS, timeout time.Duration) error {
	if s == nil {
		return nil
	}

	ueims.RecordTime = time.Now().UnixMilli()

	// 记录历史
	s.ueimsHistorys.Push(*ueims)

	// 发送
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, ueims)
}
