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
	pushOnce  sync.Once
)

func startPushWorkers() {
	if pushQueue == nil {
		pushQueue = make(chan pushJob, 100)
	}
	workers := 4
	for range workers {
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

func init() {
	pushOnce.Do(startPushWorkers)
}

// Push 尝试异步推送，如果队列满则降级为同步推送
func Push(ctx context.Context, url string, payload any) error {
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
