package utils

import (
	"sync"
)

// RingBuffer 环形缓冲区实现
// 用于高效管理固定大小的历史记录，避免频繁的内存分配和复制
type RingBuffer[T any] struct {
	data  []T
	size  int
	head  int
	tail  int
	count int
	mu    sync.RWMutex
}

// NewRingBuffer 创建指定大小的环形缓冲区
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	if size <= 0 {
		size = 1024
	}
	return &RingBuffer[T]{
		data: make([]T, size),
		size: size,
	}
}

// Push 添加元素到环形缓冲区
// 如果缓冲区已满，会覆盖最旧的元素
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.data[rb.tail] = item
	rb.tail = (rb.tail + 1) % rb.size

	if rb.count < rb.size {
		rb.count++
	} else {
		rb.head = (rb.head + 1) % rb.size
	}
}

// GetAll 获取所有元素（按插入顺序）
func (rb *RingBuffer[T]) GetAll() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return []T{}
	}

	result := make([]T, rb.count)
	if rb.head < rb.tail {
		copy(result, rb.data[rb.head:rb.tail])
	} else {
		copy(result, rb.data[rb.head:])
		copy(result[rb.size-rb.head:], rb.data[:rb.tail])
	}
	return result
}

// GetLast 获取最近的 n 个元素
// n <= 0 返回空列表，n >= count 返回所有元素
func (rb *RingBuffer[T]) GetLast(n int) []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n <= 0 || rb.count == 0 {
		return []T{}
	}

	if n >= rb.count {
		n = rb.count
	}

	result := make([]T, n)
	startIdx := (rb.tail - n + rb.size) % rb.size

	if startIdx+n <= rb.size {
		copy(result, rb.data[startIdx:startIdx+n])
	} else {
		copy(result, rb.data[startIdx:])
		copy(result[rb.size-startIdx:], rb.data[:n-(rb.size-startIdx)])
	}
	return result
}

// Count 获取当前元素数量
func (rb *RingBuffer[T]) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Resize 调整缓冲区大小
// 如果新大小小于当前元素数量，会保留最新的元素
func (rb *RingBuffer[T]) Resize(newSize int) {
	if newSize <= 0 {
		return
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	if newSize == rb.size {
		return
	}

	newData := make([]T, newSize)
	if rb.count > 0 {
		all := rb.getAllUnsafe()
		if len(all) > newSize {
			all = all[len(all)-newSize:]
		}
		copy(newData, all)
		rb.count = len(all)
		rb.head = 0
		rb.tail = rb.count
	} else {
		rb.count = 0
		rb.head = 0
		rb.tail = 0
	}

	rb.data = newData
	rb.size = newSize
}

// getAllUnsafe 内部方法，不加锁获取所有元素
func (rb *RingBuffer[T]) getAllUnsafe() []T {
	if rb.count == 0 {
		return []T{}
	}

	result := make([]T, rb.count)
	if rb.head < rb.tail {
		copy(result, rb.data[rb.head:rb.tail])
	} else {
		copy(result, rb.data[rb.head:])
		copy(result[rb.size-rb.head:], rb.data[:rb.tail])
	}
	return result
}

// Clear 清空缓冲区
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.count = 0
}
