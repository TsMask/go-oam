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
	// 创建客户端 — 使用全部配置项
	client := ws.NewClient("ws://localhost:9092/ws",
		ws.NewJSONCodec(),
		ws.WithClientSendBufferSize(2000),           // 发送队列大小
		ws.WithClientDialTimeout(10*time.Second),     // 连接超时
		ws.WithClientRequestTimeout(30*time.Second),  // 请求超时
		ws.WithClientMaxPendingRequests(5000),         // 最大 pending 数
		ws.WithClientAutoReconnect(true),              // 自动重连
		ws.WithClientMaxReconnectAttempts(5),          // 最大重连次数
	)

	// 状态回调
	client.OnState(func(state ws.State) {
		log.Printf("状态变更: %s", state)
	})

	// 错误回调
	client.OnError(func(err error) {
		log.Printf("错误: %v", err)
	})

	// 响应回调
	client.OnReceive(func(resp *ws.Response) {
		log.Printf("响应: id=%s code=%d", resp.ID, resp.Code)
	})

	// 连接
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Printf("客户端连接成功，状态: %s", client.State())

	// 同步请求
	resp, err := client.Send(ctx, &ws.Request{
		Action: "echo",
		Data:   []byte(`{"msg":"hello"}`),
	})
	if err != nil {
		log.Printf("同步请求失败: %v", err)
	} else {
		log.Printf("同步响应: id=%s code=%d data=%s", resp.ID, resp.Code, resp.Data)
	}

	// 带超时的请求
	resp, err = client.SendWithTimeout(&ws.Request{
		Action: "ping",
	}, 5*time.Second)
	if err != nil {
		log.Printf("超时请求失败: %v", err)
	} else {
		log.Printf("超时响应: id=%s code=%d data=%s", resp.ID, resp.Code, resp.Data)
	}

	// 异步请求
	client.SendAsync(ctx, &ws.Request{
		Action: "echo",
		Data:   []byte(`{"msg":"async"}`),
	}, func(resp *ws.Response, err error) {
		if err != nil {
			log.Printf("异步请求失败: %v", err)
		} else {
			log.Printf("异步响应: id=%s code=%d data=%s", resp.ID, resp.Code, resp.Data)
		}
	})

	// 并发性能测试
	log.Println("开始并发性能测试...")
	success := atomic.Int64{}
	failed := atomic.Int64{}
	start := time.Now()

	for i := 0; i < 100; i++ {
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

	time.Sleep(5 * time.Second)
	duration := time.Since(start)

	log.Printf("测试完成: 成功=%d 失败=%d 耗时=%v", success.Load(), failed.Load(), duration)
	log.Printf("QPS: %.2f", float64(success.Load())/duration.Seconds())
}
