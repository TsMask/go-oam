package core

import (
	"sync"
	"sync/atomic"
	"time"
)

// BatchScheduler 批量调度器
// 缓存写入请求，达到批量大小或超时后统一flush，提高网络传输效率
type BatchScheduler struct {
	size    int            // 批量大小
	timeout time.Duration  // 超时时间
	pending [][]byte       // 待发送队列
	mu      sync.Mutex     // 保护队列
	cond    *sync.Cond     // 条件变量
	closed  atomic.Bool    // 关闭标记
	flushFn func([][]byte) // flush回调
}

// NewBatchScheduler 创建批量调度器
// 参数：
//
//	size: 批量大小
//	timeout: 超时时间
//	flushFn: flush回调函数
func NewBatchScheduler(size int, timeout time.Duration, flushFn func([][]byte)) *BatchScheduler {
	if flushFn == nil {
		flushFn = func(batch [][]byte) {}
	}
	bs := &BatchScheduler{
		size:    size,
		timeout: timeout,
		pending: make([][]byte, 0, size),
		flushFn: flushFn,
	}
	bs.cond = sync.NewCond(&bs.mu)
	return bs
}

// Submit 提交数据到批量队列
// 参数：data 二进制数据
// 返回：成功返回true，队列已关闭返回false
func (b *BatchScheduler) Submit(data []byte) bool {
	if b.closed.Load() {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed.Load() {
		return false
	}

	b.pending = append(b.pending, data)
	if len(b.pending) >= b.size {
		b.cond.Signal()
	}
	return true
}

// Run 启动批量调度循环
// 阻塞运行，应在独立goroutine中调用
func (b *BatchScheduler) Run() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for !b.closed.Load() {
		for len(b.pending) == 0 && !b.closed.Load() {
			b.cond.Wait()
		}

		if b.closed.Load() {
			if len(b.pending) > 0 {
				batch := make([][]byte, len(b.pending))
				copy(batch, b.pending)
				b.pending = b.pending[:0]
				b.mu.Unlock()
				b.flushFn(batch)
				b.mu.Lock()
			}
			return
		}

		if len(b.pending) >= b.size {
			batch := make([][]byte, len(b.pending))
			copy(batch, b.pending)
			b.pending = b.pending[:0]
			b.mu.Unlock()
			b.flushFn(batch)
			b.mu.Lock()
			continue
		}

		now := time.Now()
		deadline := now.Add(b.timeout)
		for len(b.pending) == 0 && !b.closed.Load() {
			waitTime := time.Until(deadline)
			if waitTime <= 0 {
				break
			}
			b.cond.Wait()
		}

		if len(b.pending) > 0 {
			batch := make([][]byte, len(b.pending))
			copy(batch, b.pending)
			b.pending = b.pending[:0]
			b.mu.Unlock()
			b.flushFn(batch)
			b.mu.Lock()
		}
	}
}

// Close 关闭批量调度器
func (b *BatchScheduler) Close() {
	if b.closed.CompareAndSwap(false, true) {
		b.cond.Signal()
	}
}

// PendingCount 获取当前队列中的数据数量
func (b *BatchScheduler) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}
