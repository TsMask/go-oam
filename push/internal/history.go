package internal

import (
	"sync"
	"sync/atomic"
)

// History 统一历史记录服务
// 通过 Type 字段分表存储，所有数据类型共用同一套逻辑
type History struct {
	buffers sync.Map // type -> *ringbuffer.RingBuffer[Record]
	maxSize atomic.Int32
}

// NewHistory 创建历史记录服务
func NewHistory(maxSize int) *History {
	h := &History{}
	if maxSize > 0 {
		h.maxSize.Store(int32(maxSize))
	}
	return h
}

// getBuffer 获取指定类型的环形缓冲区
func (h *History) getBuffer(key string) *RingBuffer[Record] {
	if h == nil {
		return nil
	}

	if v, ok := h.buffers.Load(key); ok {
		return v.(*RingBuffer[Record])
	}

	maxSize := int(h.maxSize.Load())
	if maxSize <= 0 {
		maxSize = 1024 // 历史记录默认上限
	}
	newBuf := NewRingBuffer[Record](maxSize)
	actual, _ := h.buffers.LoadOrStore(key, newBuf)
	return actual.(*RingBuffer[Record])
}

// HistoryList 获取指定类型的推送历史
//
//	n < 0: 返回空
//	n == 0: 返回全部
//	n > 0: 返回最近 n 条
func (h *History) HistoryList(key string, n int) []Record {
	if h == nil || n < 0 {
		return []Record{}
	}

	buf := h.getBuffer(key)
	if buf == nil {
		return []Record{}
	}
	if n == 0 {
		return buf.GetAll()
	}
	return buf.GetLast(n)
}

// HistorySetSize 修改所有缓冲区最大记录数
func (h *History) HistorySetSize(newSize int) {
	if h == nil || newSize <= 0 {
		return
	}
	h.maxSize.Store(int32(newSize))
	h.buffers.Range(func(_, v any) bool {
		buf := v.(*RingBuffer[Record])
		buf.Resize(newSize)
		return true
	})
}

// HistorySetSizeByType 修改指定类型缓冲区最大记录数
func (h *History) HistorySetSizeByType(key string, newSize int) {
	if h == nil || key == "" || newSize <= 0 {
		return
	}
	buf := h.getBuffer(key)
	if buf != nil {
		buf.Resize(newSize)
	}
}

// HistoryClear 清除指定类型的历史记录
func (h *History) HistoryClear(key string) {
	if h == nil || key == "" {
		return
	}
	buf := h.getBuffer(key)
	if buf != nil {
		buf.Clear()
	}
}

// HistoryTypes 获取所有已有历史记录的 Type 类型列表
func (h *History) HistoryTypes() []string {
	if h == nil {
		return []string{}
	}
	types := make([]string, 0)
	h.buffers.Range(func(k, v any) bool {
		types = append(types, k.(string))
		return true
	})
	return types
}

// HistoryPush 新增历史记录
func (h *History) HistoryPush(r *Record) {
	if h == nil {
		return
	}
	buf := h.getBuffer(r.RecordType)
	if buf != nil {
		buf.Push(*r)
	}
}
