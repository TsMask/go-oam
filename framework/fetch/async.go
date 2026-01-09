package fetch

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type pushJob struct {
	url     string
	payload any
}

var (
	pushQueue chan pushJob
	workerCt  = 2
	queueSz   = 500
	initOnce  sync.Once
)

// AsyncInit 初始化异步推送配置
// 注意：必须在首次调用 AsyncPush 前调用此方法，否则使用默认配置
func AsyncInit(workerCount, queueSize int) {
	initOnce.Do(func() {
		if workerCount > 0 {
			workerCt = workerCount
		}
		if queueSize > 0 {
			queueSz = queueSize
		}
		pushQueue = make(chan pushJob, queueSz)
		startWorkers()
	})
}

func startWorkers() {
	for i := 0; i < workerCt; i++ {
		go func() {
			for job := range pushQueue {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, err := Post(job.url, Options{Ctx: ctx, JSON: job.payload})
				cancel()
				if err != nil {
					log.Printf("[OAM] push failed url: %s\n%s\n", job.url, err.Error())
				}
			}
		}()
	}
}

// AsyncPush 尝试异步POST推送，如果队列满则降级为同步推送
func AsyncPush(ctx context.Context, url string, payload any) error {
	// 确保初始化
	if pushQueue == nil {
		AsyncInit(0, 0)
	}

	select {
	case pushQueue <- pushJob{url: url, payload: payload}:
		return nil
	default:
		// 队列满，降级为同步发送
		_, err := Post(url, Options{Ctx: ctx, JSON: payload})
		if err != nil {
			return fmt.Errorf("push fallback error: %w", err)
		}
		return nil
	}
}
