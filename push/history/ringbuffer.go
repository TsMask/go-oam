package history

import (
	"sync"
)

// RingBuffer 环形缓冲区实现
//
// 设计目的:
//
//	RingBuffer 是一个固定大小的循环缓冲区，专门用于高效管理历史记录数据。
//	在推送通知场景中，它用于存储最近的消息推送记录，支持快速查询和自动淘汰旧数据。
//
// 使用场景:
//   - 消息推送系统的历史记录存储
//   - 日志缓冲区
//   - 限流计数器
//   - 需要固定窗口历史数据的任何场景
//
// 核心概念:
//
//	环形缓冲区（Ring Buffer）是一种预分配固定大小内存的数据结构，通过逻辑上的"环形"
//	索引来实现元素的循环覆盖。当缓冲区写满时，新元素会自动覆盖最旧的元素，无需
//	手动删除。这种设计避免了动态数组的内存分配开销和频繁的元素移动。
//
// 数据结构示意图:
//
//	┌─────────────────────────────────────────────────────────────────────┐
//	│                     RingBuffer (size = 8)                          │
//	│  ┌────┬────┬────┬────┬────┬────┬────┬────┐                          │
//	│  │ A  │ B  │ C  │ D  │ E  │  - │  - │  - │  (count = 5)             │
//	│  └────┴────┴────┴────┴────┴────┴────┴────┘                          │
//	│     ↑                           ↑                                  │
//	│   head (0)                   tail (5)                              │
//	│                                                                   │
//	│   head: 下一个写入位置（最新数据应该写入的位置）                    │
//	│   tail: 最旧数据位置（最早的数据所在位置）                          │
//	│                                                                   │
//	│   数据顺序: A → B → C → D → E (从旧到新)                           │
//	└─────────────────────────────────────────────────────────────────────┘
//
// 环形索引计算原理:
//
//	当索引到达缓冲区末尾时，会"绕回"到开头。
//	公式: nextIndex = (currentIndex + 1) % size
//
//	示例 (size = 8):
//	tail=5 时 Push: tail = (5 + 1) % 8 = 6
//	head=7 时 Push: head = (7 + 1) % 8 = 0  (绕回)
//
// 线程安全模型:
//   - 使用 sync.RWMutex 实现读写锁
//   - 写操作（Push, Resize, Clear）获取写锁（排他锁）
//   - 读操作（GetAll, GetLast, Count）获取读锁（共享锁）
//   - 单写者模型：建议只在一个 goroutine 中执行写操作
//   - 多读者安全：多个 goroutine 可以同时读取
//
// 时间复杂度:
//   - Push:      O(1) - 固定时间，无内存分配
//   - GetAll:    O(n) - n 为当前元素数量
//   - GetLast(n): O(n) - n 为请求获取的元素数量
//   - Count:     O(1) - 直接返回计数
//
// 空间复杂度:
//
//	O(size) - 预分配固定大小内存，不随元素数量增长
type RingBuffer[T any] struct {
	// data 底层存储数组，容量固定为 size
	data []T

	// size 缓冲区最大容量
	// 决定了环形缓冲区的物理大小，不会动态改变
	size int

	// head 头部索引，指向下一个写入位置
	// head 始终指向最新数据之后的位置（或最新数据位置，取决于实现细节）
	// 当 Push 新元素时，新元素会被写入 head 位置，然后 head 后移
	head int

	// tail 尾部索引，指向最旧数据的位置
	// tail 指向缓冲区中最早放入的元素
	// GetAll/GetLast 从 tail 开始读取数据
	tail int

	// count 当前实际存储的元素数量
	// 0 <= count <= size
	// 用于判断缓冲区是否为空或已满
	count int

	// mu 读写互斥锁
	// 写操作使用 Lock() 获取排他锁
	// 读操作使用 RLock() 获取共享锁
	mu sync.RWMutex
}

// NewRingBuffer 创建指定容量的环形缓冲区
//
// 参数:
//
//	size - 缓冲区容量，必须大于 0
//	       如果 <= 0，会自动设置为默认值 1024
//
// 返回:
//
//	初始化好的 RingBuffer 指针
//
// 示例:
//
//	rb := NewRingBuffer[string](100) // 创建容量为 100 的字符串环形缓冲区
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
//
// 行为说明:
//  1. 将元素写入 tail 当前位置
//  2. tail 向前移动一位（环形）
//  3. 如果缓冲区未满，count + 1
//     如果缓冲区已满，head 也向前移动一位（覆盖最旧的数据）
//
// 覆盖策略:
//
//	当缓冲区满时，最旧的数据会被最新的数据覆盖。
//	这是环形缓冲区的核心特性，无需手动清理旧数据。
//
// 线程安全:
//
//	使用写锁（排他锁），同一时间只允许一个写操作
//
// 时间复杂度: O(1)
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

// GetAll 获取缓冲区中的所有元素
//
// 返回顺序:
//
//	按插入顺序返回，即从最旧到最新
//
// 返回值:
//
//	包含所有元素的切片；如果缓冲区为空，返回空切片
//
// 线程安全:
//
//	使用读锁（共享锁），多个读操作可以并发执行
//
// 时间复杂度: O(n)，n 为当前元素数量
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
//
// 参数:
//
//	n - 要获取的元素数量
//	     n <= 0: 返回空切片
//	     n >= count: 返回所有元素
//
// 返回顺序:
//
//	按插入顺序返回，即从最旧到最新
//	如果请求 3 个元素，返回 [最早-2, 最早-1, 最新]
//
// 算法说明:
//
//	startIdx = (tail - n + size) % size
//	从 startIdx 开始，读取 n 个元素（环形处理）
//
// 线程安全:
//
//	使用读锁（共享锁），多个读操作可以并发执行
//
// 时间复杂度: O(n)，n 为请求获取的元素数量
//
// 示例:
//
//	假设缓冲区: [A, B, C, D, E] (A 最旧, E 最新)
//	rb.GetLast(3) -> [C, D, E]
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

// Count 获取当前缓冲区中的元素数量
//
// 返回值:
//
//	当前存储的元素数量，范围 [0, size]
//
// 线程安全:
//
//	使用读锁（共享锁）
//
// 时间复杂度: O(1)
func (rb *RingBuffer[T]) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Resize 调整缓冲区大小
//
// 行为说明:
//   - 如果新大小 == 当前大小，什么都不做
//   - 如果新大小 > 当前大小，保留所有现有数据
//   - 如果新大小 < 当前大小，保留最新的 newSize 个元素
//
// 重置索引:
//
//	调整后，head 设为 0，tail 设为新的 count 位置
//
// 线程安全:
//
//	使用写锁（排他锁）
//
// 时间复杂度: O(n)，n 为当前元素数量（需要复制数据）
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
//
// 注意事项:
//
//	此方法仅在已持有写锁的情况下调用
//	不直接对外暴露，用于 Resize 等内部操作
//
// 时间复杂度: O(n)
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
//
// 行为说明:
//
//	重置 head、tail 和 count 为初始状态
//	不实际清理底层数组中的数据（下次写入会被覆盖）
//
// 线程安全:
//
//	使用写锁（排他锁）
//
// 时间复杂度: O(1)
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.count = 0
}