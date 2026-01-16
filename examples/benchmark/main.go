package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
)

var (
	targetURL   = flag.String("url", "ws://127.0.0.1:33030/ws", "Target WebSocket URL")
	concurrency = flag.Int("c", 100, "Number of concurrent connections")
	numMsg      = flag.Int("n", 10, "Number of messages per connection")
	rate        = flag.Int("r", 1, "Messages per second per connection")
)

type Stats struct {
	ConnectTimes []time.Duration
	Latencies    []time.Duration
	SentCount    int64
	RecvCount    int64
	ErrorCount   int64
	mu           sync.Mutex
}

func (s *Stats) AddConnectTime(d time.Duration) {
	s.mu.Lock()
	s.ConnectTimes = append(s.ConnectTimes, d)
	s.mu.Unlock()
}

func (s *Stats) AddLatency(d time.Duration) {
	s.mu.Lock()
	s.Latencies = append(s.Latencies, d)
	s.mu.Unlock()
}

// # 模拟 1000 个客户端，每个发送 10 条消息，每秒 1 条
// go run examples/benchmark/main.go -c 1000 -n 10 -r 1
func main() {
	flag.Parse()

	log.Printf("Starting benchmark: URL=%s, Concurrency=%d, Msg/Conn=%d, Rate=%d/s",
		*targetURL, *concurrency, *numMsg, *rate)

	stats := &Stats{}
	var wg sync.WaitGroup
	wg.Add(*concurrency)

	startTotal := time.Now()

	// 限制连接建立速率，避免瞬间把服务端压垮
	connectRate := time.NewTicker(10 * time.Millisecond) // 每10ms建立一个连接
	defer connectRate.Stop()

	for i := 0; i < *concurrency; i++ {
		<-connectRate.C
		go runClient(i, &wg, stats)
	}

	wg.Wait()
	totalDuration := time.Since(startTotal)

	printReport(stats, totalDuration)
}

func runClient(id int, wg *sync.WaitGroup, stats *Stats) {
	defer wg.Done()

	start := time.Now()
	client := &ws.ClientConn{Url: *targetURL}
	err := client.Connect()
	if err != nil {
		log.Printf("Client %d connect error: %v", id, err)
		atomic.AddInt64(&stats.ErrorCount, 1)
		return
	}
	defer client.Close()

	connectTime := time.Since(start)
	stats.AddConnectTime(connectTime)

	done := make(chan struct{})

	// Receiver loop
	go client.ReadListen(func(err error) {
		log.Printf("Client %d read error: %v", id, err)
		atomic.AddInt64(&stats.ErrorCount, 1)
		close(done)
	}, func(_ *ws.ClientConn, messageType int, res *protocol.Response) {
		// 假设 msg 里的 Msg 字段或者其他字段可以用来关联请求
		// 这里简单处理，如果服务端返回了 PONG 或者 Echo，我们统计一次接收
		atomic.AddInt64(&stats.RecvCount, 1)
	})

	// 启动写监听
	go client.WriteListen(func(err error) {
		log.Printf("Client %d write error: %v", id, err)
		atomic.AddInt64(&stats.ErrorCount, 1)
	})

	// Sender loop
	ticker := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticker.Stop()

	for i := 0; i < *numMsg; i++ {
		select {
		case <-done:
			return
		case <-ticker.C:
			// 记录发送时间
			sendStart := time.Now()

			client.SendTextJSON(fmt.Sprintf("%d-%d", id, i), "benchmark", fmt.Sprintf("msg-%d", i))
			atomic.AddInt64(&stats.SentCount, 1)

			// 这里简单统计一下 Write 的耗时作为一部分延迟指标（非 RTT）
			// 真正的 RTT 需要收到响应后计算
			stats.AddLatency(time.Since(sendStart))
		}
	}

	// 给一点时间接收最后的响应
	time.Sleep(1 * time.Second)
	client.Close()
}

func printReport(s *Stats, totalDuration time.Duration) {
	fmt.Println("\n--- Benchmark Report ---")
	fmt.Printf("Total Duration: %v\n", totalDuration)
	fmt.Printf("Total Connections: %d\n", len(s.ConnectTimes))
	fmt.Printf("Total Sent: %d\n", atomic.LoadInt64(&s.SentCount))
	fmt.Printf("Total Recv: %d\n", atomic.LoadInt64(&s.RecvCount))
	fmt.Printf("Total Errors: %d\n", atomic.LoadInt64(&s.ErrorCount))

	if len(s.ConnectTimes) > 0 {
		sort.Slice(s.ConnectTimes, func(i, j int) bool { return s.ConnectTimes[i] < s.ConnectTimes[j] })
		fmt.Println("\nConnection Times:")
		fmt.Printf("  Min: %v\n", s.ConnectTimes[0])
		fmt.Printf("  Max: %v\n", s.ConnectTimes[len(s.ConnectTimes)-1])
		fmt.Printf("  Avg: %v\n", calculateAvg(s.ConnectTimes))
		fmt.Printf("  P99: %v\n", s.ConnectTimes[int(float64(len(s.ConnectTimes))*0.99)])
	}

	if len(s.Latencies) > 0 {
		sort.Slice(s.Latencies, func(i, j int) bool { return s.Latencies[i] < s.Latencies[j] })
		fmt.Println("\nWrite Latencies (Local):")
		fmt.Printf("  Min: %v\n", s.Latencies[0])
		fmt.Printf("  Max: %v\n", s.Latencies[len(s.Latencies)-1])
		fmt.Printf("  Avg: %v\n", calculateAvg(s.Latencies))
		fmt.Printf("  P99: %v\n", s.Latencies[int(float64(len(s.Latencies))*0.99)])
	}

	tps := float64(atomic.LoadInt64(&s.SentCount)) / totalDuration.Seconds()
	fmt.Printf("\nThroughput: %.2f msg/s\n", tps)
}

func calculateAvg(times []time.Duration) time.Duration {
	var total time.Duration
	for _, t := range times {
		total += t
	}
	return time.Duration(int64(total) / int64(len(times)))
}
