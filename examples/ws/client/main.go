package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

func main() {
	// 创建客户端（简化 API）
	client := ws.NewClient("ws://localhost:9092",
		ws.NewJSONCodec(),
		ws.WithClientSendBufferSize(1000),
		ws.WithClientWorkers(4),
		ws.WithClientBatchSize(100),
		ws.WithClientRateLimit(10000),
		ws.WithClientDialTimeout(30*time.Second),
	)

	// 连接
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	// 状态回调（使用导出的 State 类型）
	client.OnState(func(state ws.State) {
		log.Printf("状态变更: %s", state)
	})

	// 错误处理
	client.OnError(func(err error) {
		log.Printf("错误: %v", err)
	})

	// 响应回调（使用导出的 Response 类型）
	client.OnReceive(func(resp *ws.Response) {
		log.Printf("响应: id=%s code=%d", resp.ID, resp.Code)
	})

	log.Println("客户端连接成功")

	// 性能测试
	success := atomic.Int64{}
	failed := atomic.Int64{}
	start := time.Now()

	// 1000请求性能测试
	for i := 0; i < 1000; i++ {
		go func(seq int) {
			_, err := client.Send(ctx, &ws.Request{
				Action: "echo",
				Data:   []byte(fmt.Sprintf("ping-%d", seq)),
			})
			if err != nil {
				failed.Add(1)
			} else {
				success.Add(1)
			}
		}(i)
	}

	// 等待完成
	time.Sleep(5 * time.Second)
	duration := time.Since(start)
	metrics := client.Metrics()

	log.Printf("测试完成: 成功=%d 失败=%d 耗时=%v", success.Load(), failed.Load(), duration)
	log.Printf("QPS: %.2f", float64(success.Load())/duration.Seconds())
	log.Printf("指标: 队列=%d 活跃=%d 限流=%d 背压=%d",
		metrics.SendQueueSize.Load(),
		metrics.ActiveRequests.Load(),
		metrics.RateLimitDrops.Load(),
		metrics.BackpressureHits.Load(),
	)

	// 测试异步请求
	log.Println("测试异步请求...")
	client.SendAsync(ctx, &ws.Request{
		Action: "echo",
		Data:   []byte(`{"test":"async"}`),
	}, func(resp *ws.Response, err error) {
		if err != nil {
			log.Printf("异步请求失败: %v", err)
		} else {
			log.Printf("异步请求成功: id=%s code=%d", resp.ID, resp.Code)
		}
	})

	time.Sleep(1 * time.Second)
}
