package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

// go run main.go
func main() {
	// 创建客户端 — 默认 JSON 编解码，与服务端匹配
	client := ws.NewClient("ws://localhost:9092/ws",
		ws.WithClientCodec("protobuf"),
		ws.WithClientDialTimeout(10*time.Second),
		ws.WithClientAutoReconnect(false),
		ws.WithClientHeartbeat(15*time.Second),
	)

	var recvCount atomic.Int64
	var wg sync.WaitGroup

	// 状态回调
	client.OnState(func(state ws.State) {
		log.Printf("状态变更: %s", state)
	})

	// 错误回调
	client.OnError(func(err error) {
		log.Printf("错误: %v", err)
	})

	// 响应回调 — 所有响应异步到达，按 ID 匹配请求
	client.OnReceive(func(resp *ws.Response) {
		recvCount.Add(1)
		dataStr := ""
		if resp.Data != nil {
			dataStr = string(resp.Data)
			if len(dataStr) > 80 {
				dataStr = dataStr[:80] + "..."
			}
		}
		log.Printf("[收到] id=%s action=%s code=%d data=%s", resp.ID, resp.Action, resp.Code, dataStr)
		wg.Done()
	})

	// 连接
	if err := client.Connect(context.Background()); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Printf("客户端连接成功，状态: %s", client.State())

	// === 基础测试 ===
	log.Println("=== 基础测试 ===")
	wg.Add(3)

	client.Send(&ws.Request{Action: "echo", Data: json.RawMessage(`"hello"`)})
	client.Send(&ws.Request{Action: "ping"})
	client.Send(&ws.Request{Action: "info"})

	// 等待基础测试响应
	if waitWithTimeout(&wg, 5*time.Second) {
		log.Println("基础测试响应超时")
	}
	log.Printf("基础测试完成，收到 %d 条响应\n", recvCount.Load())

	// === 并发性能测试 ===
	log.Println("=== 并发性能测试 ===")
	recvCount.Store(0)

	total := 10
	wg.Add(total)

	var SendData, sendFail atomic.Int64
	start := time.Now()

	for i := 0; i < total; i++ {
		go func(seq int) {
			err := client.Send(&ws.Request{
				Action: "echo",
				Data:   json.RawMessage(fmt.Sprintf(`{"seq":%d,"msg":"perf-%d"}`, seq, seq)),
			})
			if err != nil {
				sendFail.Add(1)
				wg.Done() // 发送失败也要 Done，否则永远等不到
			} else {
				SendData.Add(1)
			}
		}(i)
	}

	// 等待所有响应到达（发送 + 接收 的完整往返）
	if waitWithTimeout(&wg, 30*time.Second) {
		log.Printf("性能测试超时，已收到 %d/%d 条响应", recvCount.Load(), total)
	}

	duration := time.Since(start)
	success := recvCount.Load()
	qps := float64(success) / duration.Seconds()
	log.Printf("性能测试完成: 发送成功=%d 失败=%d 响应=%d 耗时=%v QPS=%.1f",
		SendData.Load(), sendFail.Load(), success, duration.Round(time.Millisecond), qps)
}

// waitWithTimeout 等待 WaitGroup 完成或超时
// 返回 true 表示超时
func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return false
	case <-time.After(timeout):
		return true
	}
}
