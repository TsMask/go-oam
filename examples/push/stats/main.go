// Push 模块性能基准测试
//
// 测试 push 模块各组件在高并发下的 QPS、延迟分布和数据正确性：
//  1. Metrics 指标采集（标准 / 分片对比）
//  2. History 历史记录（标准 / 分片对比）
//  3. HTTP Client（同步 / 异步 / 批量 / 重试降级）
//  4. Timer 定时器并发
//  5. 综合场景（定时采集 → 异步推送）
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/push"
	"github.com/tsmask/go-oam/push/client"
)

// ============================================================================
// 配置
// ============================================================================

var (
	concurrency = 50              // 并发协程数
	testDur     = 3 * time.Second // HTTP 测试持续时长
)

// ============================================================================
// 统计收集器
// ============================================================================

type benchResult struct {
	name      string
	totalOps  int64
	errors    int64
	wallDur   time.Duration // 挂钟时间
	latencies []time.Duration
	peakGor   int
	extra     string
}

type collector struct {
	latMu     sync.Mutex
	latencies []time.Duration
	ops       atomic.Int64
	errors    atomic.Int64
	gorMu     sync.Mutex
	peakGor   int
	stopCh    chan struct{}
	wg        sync.WaitGroup
	wallStart time.Time
}

func newCollector() *collector {
	c := &collector{
		latencies: make([]time.Duration, 0, 65536),
		stopCh:    make(chan struct{}),
		wallStart: time.Now(),
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				g := runtime.NumGoroutine()
				c.gorMu.Lock()
				if g > c.peakGor {
					c.peakGor = g
				}
				c.gorMu.Unlock()
			case <-c.stopCh:
				return
			}
		}
	}()
	return c
}

func (c *collector) record(d time.Duration) {
	c.ops.Add(1)
	c.latMu.Lock()
	c.latencies = append(c.latencies, d)
	c.latMu.Unlock()
}

func (c *collector) recordError() { c.ops.Add(1); c.errors.Add(1) }

func (c *collector) finish(name, extra string) benchResult {
	close(c.stopCh)
	c.wg.Wait()

	lats := c.latencies
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })

	r := benchResult{
		name:      name,
		totalOps:  c.ops.Load(),
		errors:    c.errors.Load(),
		wallDur:   time.Since(c.wallStart),
		latencies: lats,
		extra:     extra,
	}
	c.gorMu.Lock()
	r.peakGor = c.peakGor
	c.gorMu.Unlock()
	return r
}

func (r *benchResult) qps() float64 {
	if r.wallDur == 0 {
		return 0
	}
	return float64(len(r.latencies)) / r.wallDur.Seconds()
}

func (r *benchResult) percentile(p float64) time.Duration {
	if len(r.latencies) == 0 {
		return 0
	}
	idx := int(float64(len(r.latencies)-1) * p / 100)
	return r.latencies[idx]
}

func (r *benchResult) avg() time.Duration {
	if len(r.latencies) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range r.latencies {
		sum += d
	}
	return sum / time.Duration(len(r.latencies))
}

func (r *benchResult) successRate() float64 {
	if r.totalOps == 0 {
		return 0
	}
	return float64(r.totalOps-r.errors) / float64(r.totalOps) * 100
}

// ============================================================================
// 输出
// ============================================================================

func printResult(r benchResult) {
	fmt.Printf("  %-28s  ops=%-8d  qps=%-12.0f  avg=%-10s  p50=%-10s  p99=%-10s  gor=%-4d  err=%d  %.1f%%\n",
		r.name, r.totalOps, r.qps(), r.avg(),
		r.percentile(50), r.percentile(99), r.peakGor, r.errors, r.successRate())
	if r.extra != "" {
		fmt.Printf("    → %s\n", r.extra)
	}
}

// ============================================================================
// 1. Metrics 指标采集基准
// ============================================================================

func benchMetrics(standard bool) benchResult {
	label := "Metrics"
	if !standard {
		label = "ShardedMetrics"
	}

	c := newCollector()
	var wg sync.WaitGroup
	opsPer := 50000

	barrier := make(chan struct{})

	if standard {
		m := push.NewMetrics()
		for i := 0; i < 100; i++ {
			m.Register(fmt.Sprintf("m_%d", i), 0, 1, 0, 1e15)
		}
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-barrier
				for j := 0; j < opsPer; j++ {
					t0 := time.Now()
					name := fmt.Sprintf("m_%d", id%100)
					switch j % 4 {
					case 0:
						m.Inc(name)
					case 1:
						m.Dec(name)
					case 2:
						m.IncBy(name, 2.5)
					case 3:
						m.Get(name)
					}
					c.record(time.Since(t0))
				}
			}(i)
		}
	} else {
		sm := push.NewShardedMetrics()
		for i := 0; i < 100; i++ {
			sm.Register(fmt.Sprintf("m_%d", i), 0, 1, 0, 1e15)
		}
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-barrier
				for j := 0; j < opsPer; j++ {
					t0 := time.Now()
					name := fmt.Sprintf("m_%d", id%100)
					switch j % 4 {
					case 0:
						sm.Inc(name)
					case 1:
						sm.Dec(name)
					case 2:
						sm.IncBy(name, 2.5)
					case 3:
						sm.Get(name)
					}
					c.record(time.Since(t0))
				}
			}(i)
		}
	}

	close(barrier) // 同时开始
	wg.Wait()
	return c.finish(label, fmt.Sprintf("%d 协程 × %d ops, 100 指标, Inc/Dec/IncBy/Get", concurrency, opsPer))
}

// ============================================================================
// 2. History 历史记录基准
// ============================================================================

func benchHistory(sharded bool) benchResult {
	label := "History"
	if sharded {
		label = "ShardedHistory"
	}

	c := newCollector()
	var wg sync.WaitGroup
	opsPer := 50000
	barrier := make(chan struct{})

	type item struct {
		Seq  int
		Data string
	}

	if !sharded {
		h := push.NewHistory[item](10240)
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-barrier
				for j := 0; j < opsPer; j++ {
					t0 := time.Now()
					key := fmt.Sprintf("k_%d", id%20)
					h.Push(key, item{Seq: id*opsPer + j, Data: key})
					if j%10 == 0 {
						h.List(key, 50)
					}
					c.record(time.Since(t0))
				}
			}(i)
		}
	} else {
		sh := push.NewShardedHistory[item](10240)
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-barrier
				for j := 0; j < opsPer; j++ {
					t0 := time.Now()
					key := fmt.Sprintf("k_%d", id%20)
					sh.Push(key, item{Seq: id*opsPer + j, Data: key})
					if j%10 == 0 {
						sh.List(key, 50)
					}
					c.record(time.Since(t0))
				}
			}(i)
		}
	}

	close(barrier)
	wg.Wait()
	return c.finish(label, fmt.Sprintf("%d 协程 × %d ops, 20 key, Push + List(50)/10", concurrency, opsPer))
}

// ============================================================================
// 3. HTTP Client 基准
// ============================================================================

func benchHTTPSync(serverURL string) benchResult {
	cli := client.New(client.WithTimeout(5 * time.Second))
	defer cli.Close()

	c := newCollector()
	var wg sync.WaitGroup
	opsPer := 200

	barrier := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-barrier
			for j := 0; j < opsPer; j++ {
				t0 := time.Now()
				err := cli.Push(serverURL, map[string]any{"id": id*opsPer + j})
				if err != nil {
					c.recordError()
				} else {
					c.record(time.Since(t0))
				}
			}
		}(i)
	}
	close(barrier)
	wg.Wait()
	return c.finish("Client.sync", "")
}

func benchHTTPAsync(serverURL string) benchResult {
	cli := client.New(client.WithAsyncQueue(concurrency, 8192), client.WithTimeout(5*time.Second))
	defer cli.Close()

	c := newCollector()
	var wg sync.WaitGroup
	opsPer := 500

	barrier := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-barrier
			for j := 0; j < opsPer; j++ {
				t0 := time.Now()
				err := cli.AsyncPush(serverURL, map[string]any{"id": id*opsPer + j})
				if err != nil {
					c.recordError()
				} else {
					c.record(time.Since(t0))
				}
			}
		}(i)
	}
	close(barrier)
	wg.Wait()
	time.Sleep(500 * time.Millisecond) // 等异步投递完成

	stats := cli.Stats()
	extra := fmt.Sprintf("Workers=%d Processed=%d Failed=%d", stats.ActiveWorkers, stats.TotalProcessed, stats.FailedCount)
	return c.finish("Client.async", extra)
}

func benchHTTPBatch(serverURL string) benchResult {
	cli := client.New(client.WithAsyncQueue(concurrency, 8192), client.WithTimeout(5*time.Second))
	defer cli.Close()

	c := newCollector()
	var wg sync.WaitGroup
	opsPer := 100 // 每次 BatchPush 10 条，共 1000 条/协程

	barrier := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-barrier
			for j := 0; j < opsPer; j++ {
				payloads := make([]any, 10)
				for k := 0; k < 10; k++ {
					payloads[k] = map[string]any{"id": id*opsPer*10 + j*10 + k}
				}
				t0 := time.Now()
				err := cli.BatchPush(serverURL, payloads)
				if err != nil {
					c.recordError()
				} else {
					c.record(time.Since(t0))
				}
			}
		}(i)
	}
	close(barrier)
	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	stats := cli.Stats()
	extra := fmt.Sprintf("Workers=%d Processed=%d", stats.ActiveWorkers, stats.TotalProcessed)
	return c.finish("Client.batch", extra)
}

func benchRetry() benchResult {
	var reqCount atomic.Int64
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := reqCount.Add(1)
		if n%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer failServer.Close()

	cli := client.New(client.WithRetry(2), client.WithTimeout(5*time.Second))
	defer cli.Close()

	c := newCollector()
	var wg sync.WaitGroup
	opsPer := 100

	barrier := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-barrier
			for j := 0; j < opsPer; j++ {
				t0 := time.Now()
				err := cli.Push(failServer.URL, map[string]any{"id": id*opsPer + j})
				if err != nil {
					c.recordError()
				} else {
					c.record(time.Since(t0))
				}
			}
		}(i)
	}
	close(barrier)
	wg.Wait()

	reqTotal := reqCount.Load()
	extra := fmt.Sprintf("总请求 %d（含重试）, 重试 2 次, 50%% 失败率", reqTotal)
	return c.finish("Client.retry", extra)
}

// ============================================================================
// 4. Timer 并发基准
// ============================================================================

func benchTimer() benchResult {
	var callbackTotal atomic.Int64
	var wg sync.WaitGroup
	timerCount := concurrency

	start := time.Now()
	for i := 0; i < timerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			timer := push.NewTimer()
			timer.Start(10*time.Millisecond, func(t time.Time) {
				callbackTotal.Add(1)
			})
			time.Sleep(testDur)
			timer.Stop()
		}()
	}
	wg.Wait()
	wallDur := time.Since(start)

	totalCallbacks := callbackTotal.Load()
	extra := fmt.Sprintf("%d 个 Timer 并发, %s, 总回调 %d 次, 回调QPS=%.0f",
		timerCount, wallDur.Round(time.Millisecond), totalCallbacks, float64(totalCallbacks)/wallDur.Seconds())

	return benchResult{
		name:     "Timer",
		totalOps: totalCallbacks,
		wallDur:  wallDur,
		extra:    extra,
	}
}

// ============================================================================
// 5. 综合场景：定时采集 + 异步推送
// ============================================================================

func benchIntegration(serverURL string) benchResult {
	sm := push.NewShardedMetrics()
	for i := 0; i < 10; i++ {
		sm.Register(fmt.Sprintf("metric_%d", i), 0, 1, 0, 1e9)
	}
	sm.Register("pushed", 0, 1, 0, 1e9)
	sm.Register("failed", 0, 1, 0, 1e9)

	p := push.New(push.WithBaseURL(serverURL), push.WithTimeout(3*time.Second))

	c := newCollector()

	timer := push.NewTimer()
	timer.Start(20*time.Millisecond, func(t time.Time) {
		for i := 0; i < 10; i++ {
			sm.IncBy(fmt.Sprintf("metric_%d", i), float64(i+1))
		}
		delta := sm.Flush()
		data, _ := json.Marshal(delta)

		t0 := time.Now()
		err := p.SendAsync(&push.Record{
			NeUID:      "ne-bench",
			RecordType: "integration",
			RecordData: data,
		}, nil)
		if err != nil {
			sm.Inc("failed")
		} else {
			sm.Inc("pushed")
		}
		c.record(time.Since(t0))
	})

	time.Sleep(testDur)
	timer.Stop()
	time.Sleep(300 * time.Millisecond)
	p.Close()

	snap := sm.Snapshot()
	extra := fmt.Sprintf("推送 %d 次, 失败 %d",
		int64(snap["pushed"]), int64(snap["failed"]))
	return c.finish("Integration", extra)
}

// ============================================================================
// 6. 正确性验证
// ============================================================================

func verifyCorrectness() {
	fmt.Println("\n  ── 正确性验证 ──")
	passed := 0
	failed := 0

	check := func(cond bool, msg string) {
		if cond {
			passed++
			fmt.Printf("  [PASS] %s\n", msg)
		} else {
			failed++
			fmt.Printf("  [FAIL] %s\n", msg)
		}
	}

	// ShardedMetrics 并发 Inc
	sm := push.NewShardedMetrics()
	sm.Register("v", 0, 1, 0, 1e15)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				sm.Inc("v")
			}
		}()
	}
	wg.Wait()
	expected := int64(concurrency) * 10000
	check(int64(sm.Get("v")) == expected, fmt.Sprintf("ShardedMetrics 并发 Inc: 期望=%d 实际=%.0f", expected, sm.Get("v")))

	// Metrics Flush 增量
	m := push.NewMetrics()
	m.Register("f", 100, 1, 0, 1e15)
	m.Inc("f")
	m.Inc("f")
	delta := m.Flush()
	check(delta["f"] == 2 && m.GetDelta("f") == 0, fmt.Sprintf("Metrics Flush 增量: delta=%.0f post=%.0f", delta["f"], m.GetDelta("f")))

	// History 环形覆盖
	h := push.NewHistory[int](5)
	for i := 0; i < 8; i++ {
		h.Push("t", i)
	}
	items := h.List("t", 0)
	check(len(items) == 5 && items[0] == 3 && items[4] == 7, fmt.Sprintf("History 环形覆盖: len=%d [%d..%d]", len(items), items[0], items[len(items)-1]))

	// ShardedHistory BatchPush
	type s struct{ ID int }
	sh := push.NewShardedHistory[s](200)
	recs := make([]s, 100)
	for i := range recs {
		recs[i] = s{ID: i}
	}
	sh.BatchPush(func(r s) string {
		if r.ID < 50 {
			return "a"
		}
		return "b"
	}, recs)
	check(sh.CountAll() == 100, fmt.Sprintf("ShardedHistory BatchPush: total=%d", sh.CountAll()))

	fmt.Printf("\n  正确性: %d 通过, %d 失败\n", passed, failed)
}

// ============================================================================
// main
// ============================================================================

func main() {
	debug.SetGCPercent(200)

	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &concurrency)
	}
	if concurrency <= 0 {
		concurrency = 50
	}

	fmt.Println()
	fmt.Printf("  Push 模块性能基准测试    并发=%d  CPU=%d\n", concurrency, runtime.NumCPU())
	fmt.Println()

	startAll := time.Now()
	var results []benchResult

	// ── Metrics ──
	fmt.Println("  [Metrics 指标采集]")
	r1 := benchMetrics(true)
	printResult(r1)
	results = append(results, r1)
	runtime.GC()

	r2 := benchMetrics(false)
	printResult(r2)
	results = append(results, r2)
	runtime.GC()

	improve := 0.0
	if r1.qps() > 0 {
		improve = (r2.qps() - r1.qps()) / r1.qps() * 100
	}
	fmt.Printf("    → Sharded vs Standard: %+.1f%%  (p99: %s → %s)\n\n", improve, r1.percentile(99), r2.percentile(99))

	// ── History ──
	fmt.Println("  [History 历史记录]")
	r3 := benchHistory(false)
	printResult(r3)
	results = append(results, r3)
	runtime.GC()

	r4 := benchHistory(true)
	printResult(r4)
	results = append(results, r4)
	runtime.GC()

	improveH := 0.0
	if r3.qps() > 0 {
		improveH = (r4.qps() - r3.qps()) / r3.qps() * 100
	}
	fmt.Printf("    → Sharded vs Standard: %+.1f%%  (p99: %s → %s)\n\n", improveH, r3.percentile(99), r4.percentile(99))

	// ── HTTP Client ──
	fmt.Println("  [HTTP Client]")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r5 := benchHTTPSync(server.URL)
	printResult(r5)
	results = append(results, r5)

	r6 := benchHTTPAsync(server.URL)
	printResult(r6)
	results = append(results, r6)

	r7 := benchHTTPBatch(server.URL)
	printResult(r7)
	results = append(results, r7)

	// ── 重试 ──
	fmt.Println()
	r8 := benchRetry()
	printResult(r8)
	results = append(results, r8)

	// ── Timer ──
	fmt.Println("\n  [Timer 定时器]")
	r9 := benchTimer()
	fmt.Printf("  %-28s  callbacks=%-8d  回调QPS=%-12.0f  %s\n",
		r9.name, r9.totalOps, float64(r9.totalOps)/r9.wallDur.Seconds(), r9.extra)
	results = append(results, r9)

	// ── 综合 ──
	fmt.Println("\n  [综合场景]")
	r10 := benchIntegration(server.URL)
	printResult(r10)
	results = append(results, r10)

	// ── 正确性 ──
	verifyCorrectness()

	// ── 汇总 ──
	totalDur := time.Since(startAll)
	fmt.Println()
	fmt.Printf("  %-28s  %-10s  %-12s  %-10s  %-10s  %-5s\n",
		"测试项", "ops", "qps", "p50", "p99", "成功率")
	fmt.Println("  " + "─")
	for _, r := range results {
		fmt.Printf("  %-28s  %-10d  %-12.0f  %-10s  %-10s  %.1f%%\n",
			r.name, r.totalOps, math.Round(r.qps()),
			r.percentile(50), r.percentile(99), r.successRate())
	}
	fmt.Printf("\n  总耗时: %s\n\n", totalDur.Round(time.Millisecond))
}
