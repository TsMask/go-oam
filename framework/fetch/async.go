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
		pushQueue = make(chan pushJob, 1024)
	}
	workers := 4
	for range workers {
		go func() {
			for job := range pushQueue {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, err := PostJSON(ctx, job.url, job.payload, nil)
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

func EnqueuePush(url string, payload any) error {
	select {
	case pushQueue <- pushJob{url: url, payload: payload}:
		return nil
	default:
		return fmt.Errorf("push queue full")
	}
}
