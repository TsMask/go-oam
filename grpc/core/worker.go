package core

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

// WorkerPool goroutine工作池
// 复用goroutine，减少创建销毁开销
type WorkerPool struct {
	workers    int         // Worker数量
	jobCh      chan func() // 任务通道
	wg         sync.WaitGroup
	activeJobs atomic.Int64 // 活跃任务数
	queuedJobs atomic.Int64 // 队列任务数
	closed     atomic.Bool  // 关闭标记
}

// NewWorkerPool 创建工作池
// 参数：
//
//	workers: Worker数量
//	queueSize: 队列大小
func NewWorkerPool(workers, queueSize int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	if queueSize <= 0 {
		queueSize = workers * 10
	}

	wp := &WorkerPool{
		workers: workers,
		jobCh:   make(chan func(), queueSize),
	}

	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobCh {
		if job != nil {
			wp.activeJobs.Add(1)
			func() {
				defer wp.activeJobs.Add(-1)
				job()
			}()
		}
	}
}

// Submit 提交任务
// 参数：job 任务函数
// 返回：提交成功返回true，失败返回false
func (wp *WorkerPool) Submit(job func()) bool {
	if wp.closed.Load() {
		return false
	}

	select {
	case wp.jobCh <- job:
		wp.queuedJobs.Add(1)
		return true
	default:
		return false
	}
}

// SubmitWait 提交任务并等待完成
// 参数：job 任务函数
// 返回：提交成功返回true，失败返回false
func (wp *WorkerPool) SubmitWait(job func()) bool {
	if wp.closed.Load() {
		return false
	}

	done := make(chan struct{}, 1)
	select {
	case wp.jobCh <- func() {
		job()
		done <- struct{}{}
	}:
		wp.queuedJobs.Add(1)
		<-done
		return true
	default:
		return false
	}
}

// SubmitContext 提交带上下文的异步任务
// 参数：
//
//	ctx: 上下文
//	job: 任务函数
//
// 返回：提交成功返回true，失败返回false
func (wp *WorkerPool) SubmitContext(ctx context.Context, job func()) bool {
	if wp.closed.Load() {
		return false
	}

	done := make(chan struct{}, 1)
	select {
	case wp.jobCh <- func() {
		job()
		select {
		case done <- struct{}{}:
		default:
		}
	}:
		wp.queuedJobs.Add(1)
		select {
		case <-done:
		case <-ctx.Done():
			return false
		}
		return true
	default:
		return false
	}
}

// Close 关闭工作池
func (wp *WorkerPool) Close() {
	if wp.closed.CompareAndSwap(false, true) {
		close(wp.jobCh)
	}
}

// Wait 等待所有Worker完成
func (wp *WorkerPool) Wait() {
	wp.Close()
	wp.wg.Wait()
}

// ActiveWorkers 获取活跃Worker数
func (wp *WorkerPool) ActiveWorkers() int {
	return int(wp.activeJobs.Load())
}

// QueuedJobs 获取队列任务数
func (wp *WorkerPool) QueuedJobs() int {
	return int(wp.queuedJobs.Load())
}

// TotalWorkers 获取Worker总数
func (wp *WorkerPool) TotalWorkers() int {
	return wp.workers
}

// QueueCapacity 获取队列容量
func (wp *WorkerPool) QueueCapacity() int {
	return cap(wp.jobCh)
}

// IsClosed 检查是否已关闭
func (wp *WorkerPool) IsClosed() bool {
	return wp.closed.Load()
}
