package fetch

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// pushJob 异步推送任务
type pushJob struct {
	ctx     context.Context // 请求上下文（超时/取消）
	url     string          // 推送地址
	payload any             // JSON Body
}

var (
	pushQueue chan pushJob
	workerCt  = 2
	queueSz   = 500
	initOnce  sync.Once
	closeOnce sync.Once
	doneCh    chan struct{} // worker 退出信号
)

// AsyncInit 初始化异步推送队列
// workerCount: 工作协程数，默认 2
// queueSize: 队列缓冲大小，默认 500
// 必须在首次 AsyncPush 前调用，否则使用默认值
func AsyncInit(workerCount, queueSize int) {
	initOnce.Do(func() {
		if workerCount > 0 {
			workerCt = workerCount
		}
		if queueSize > 0 {
			queueSz = queueSize
		}
		pushQueue = make(chan pushJob, queueSz)
		doneCh = make(chan struct{}, workerCt)
		startWorkers()
	})
}

// startWorkers 启动消费协程
func startWorkers() {
	for i := 0; i < workerCt; i++ {
		go func() {
			defer func() { doneCh <- struct{}{} }()
			for job := range pushQueue {
				opts := Options{JSON: job.payload}
				if job.ctx != nil {
					opts.Ctx = job.ctx
				}
				if _, err := Post(job.url, opts); err != nil {
					log.Printf("[OAM] push failed %s: %s\n", job.url, err.Error())
				}
			}
		}()
	}
}

// AsyncClose 优雅关闭异步推送队列
// 关闭后不再接受新任务，等待已有任务处理完毕
func AsyncClose() {
	closeOnce.Do(func() {
		if pushQueue != nil {
			close(pushQueue)
			for i := 0; i < workerCt; i++ {
				<-doneCh
			}
		}
	})
}

// AsyncPush 异步 POST 推送
// 队列未满时异步投递，队列满时降级为同步发送
func AsyncPush(ctx context.Context, url string, payload any) error {
	// sync.Once 保证只初始化一次，此处做快速路径避免无谓函数调用
	if pushQueue == nil {
		AsyncInit(0, 0)
	}

	select {
	case pushQueue <- pushJob{ctx: ctx, url: url, payload: payload}:
		return nil
	default:
		// 队列满，降级同步发送
		_, err := Post(url, Options{Ctx: ctx, JSON: payload})
		if err != nil {
			return fmt.Errorf("push fallback: %w", err)
		}
		return nil
	}
}
