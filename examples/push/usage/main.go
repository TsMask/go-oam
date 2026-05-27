// 推送模块全功能测试
//
// 本地启动 mock HTTP 服务，测试 push 模块所有核心功能：
//  1. Record 消息发送（同步/异步/自定义参数）
//  2. HTTP Client（Worker 池、队列、批量、重试、降级）
//  3. Metrics 指标采集（标准/分片对比）
//  4. History 历史记录（标准/分片、批量、Resize）
//  5. Timer 定时器
//  6. 综合场景（定时采集 → 指标聚合 → 异步推送）
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/push"
	"github.com/tsmask/go-oam/push/client"
)

// ============================================================================
// Mock 服务
// ============================================================================

// mockHandler 记录收到的请求
type mockHandler struct {
	received atomic.Int64
	byType   sync.Map // map[string]*atomic.Int64
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (h *mockHandler) countByType(t string) *atomic.Int64 {
	v, _ := h.byType.LoadOrStore(t, &atomic.Int64{})
	return v.(*atomic.Int64)
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	var record map[string]any
	if err := json.Unmarshal(body, &record); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	recordType, _ := record["record_type"].(string)
	h.received.Add(1)
	h.countByType(recordType).Add(1)

	w.WriteHeader(http.StatusOK)
}

func (h *mockHandler) total() int64         { return h.received.Load() }
func (h *mockHandler) typed(t string) int64 { return h.countByType(t).Load() }

// ============================================================================
// 测试结果
// ============================================================================

var passed int
var failed int

func assert(cond bool, msg string) {
	if cond {
		passed++
		fmt.Printf("  PASS  %s\n", msg)
	} else {
		failed++
		fmt.Printf("  FAIL  %s\n", msg)
	}
}

func section(name string) {
	fmt.Printf("\n[%s]\n", name)
}

// ============================================================================
// 1. Record 推送测试
// ============================================================================

func testRecordPush(handler *mockHandler, serverURL string) {
	section("1. Record 消息推送")

	p := push.New(
		push.WithBaseURL(serverURL),
		push.WithPushURI("/api/push/receive"),
		push.WithTimeout(5*time.Second),
	)
	defer p.Close()

	// 同步发送
	before := handler.total()
	record := &push.Record{
		CoreUID:    "core-001",
		NeUID:      "ne-001",
		RecordType: "alarm",
		RecordData: json.RawMessage(`{"level":"critical","message":"CPU overload"}`),
		Params:     map[string]string{"source": "agent"},
	}
	err := p.Send(record, nil)
	assert(err == nil, "同步发送 alarm 成功")
	assert(handler.total() == before+1, "服务端收到 1 条消息")
	assert(record.RecordTime > 0, "RecordTime 自动填充")

	// 多种 record_type
	types := []string{"metrics", "kpi", "cdr", "nb_state"}
	for _, t := range types {
		data, _ := json.Marshal(map[string]any{"value": 42.5})
		err = p.Send(&push.Record{
			NeUID:      "ne-001",
			RecordType: t,
			RecordData: data,
		}, nil)
		assert(err == nil, fmt.Sprintf("同步发送 %s 成功", t))
	}
	assert(handler.total() == before+5, fmt.Sprintf("服务端共收到 %d 条", 5))

	// 异步发送 100 条
	before = handler.total()
	for i := 0; i < 100; i++ {
		data, _ := json.Marshal(map[string]any{"index": i})
		err = p.SendAsync(&push.Record{
			NeUID:      fmt.Sprintf("ne-%03d", i%10),
			RecordType: "async_test",
			RecordData: data,
		}, nil)
		assert(err == nil, fmt.Sprintf("异步发送第 %d 条入队成功", i+1))
	}
	time.Sleep(500 * time.Millisecond)
	assert(handler.total() >= before+100, fmt.Sprintf("异步消息已投递 (收到 %d/%d)", handler.total()-before, 100))

	// 自定义 URL + 超时
	before = handler.total()
	err = p.Send(&push.Record{
		RecordType: "custom_url",
		RecordData: json.RawMessage(`{"test":"send_params"}`),
	}, &push.SendParams{
		URL:     serverURL + "/api/push/receive",
		Timeout: 2 * time.Second,
	})
	assert(err == nil, "自定义 URL 和超时发送成功")
	assert(handler.total() == before+1, "自定义参数消息已收到")

	// 空 RecordData
	err = p.Send(&push.Record{
		NeUID:      "ne-001",
		RecordType: "empty_data",
	}, nil)
	assert(err == nil, "空 RecordData 发送成功")

	// 按类型统计
	fmt.Println("\n  --- 按类型统计 ---")
	for _, t := range []string{"alarm", "metrics", "kpi", "cdr", "nb_state", "async_test", "custom_url", "empty_data"} {
		count := handler.typed(t)
		if count > 0 {
			fmt.Printf("  TYPE  %-12s %d\n", t, count)
		}
	}
}

// ============================================================================
// 2. Client HTTP 客户端测试
// ============================================================================

func testClient(_ *mockHandler, serverURL string) {
	section("2. HTTP Client")

	cli := client.New(
		client.WithBaseURL(serverURL),
		client.WithTimeout(5*time.Second),
		client.WithRetry(2),
		client.WithWorkers(4),
		client.WithQueueSize(256),
	)
	defer cli.Close()

	// 基础同步推送
	err := cli.Push(serverURL+"/api/push/receive", map[string]any{
		"record_type": "client_sync",
		"ne_uid":      "ne-001",
	})
	assert(err == nil, "Client.Push 同步发送成功")

	// 自定义超时同步推送
	err = cli.PushTimeout(serverURL+"/api/push/receive", map[string]any{
		"record_type": "client_timeout",
	}, 3*time.Second)
	assert(err == nil, "Client.PushTimeout 成功")

	// 异步推送 50 条
	for i := 0; i < 50; i++ {
		cli.AsyncPush(serverURL+"/api/push/receive", map[string]any{
			"record_type": "client_async",
			"index":       i,
		})
	}
	time.Sleep(300 * time.Millisecond)

	// 异步自定义超时
	err = cli.AsyncPushTimeout(serverURL+"/api/push/receive", map[string]any{
		"record_type": "client_async_timeout",
	}, 2*time.Second)
	assert(err == nil, "Client.AsyncPushTimeout 成功")
	time.Sleep(200 * time.Millisecond)

	// 批量推送
	payloads := make([]any, 20)
	for i := range payloads {
		payloads[i] = map[string]any{
			"record_type": "client_batch",
			"batch_index": i,
		}
	}
	err = cli.BatchPush(serverURL+"/api/push/receive", payloads)
	assert(err == nil, "Client.BatchPush 20 条成功")

	// 运行状态
	stats := cli.Stats()
	fmt.Printf("  PoolStats: ActiveWorkers=%d Processed=%d Failed=%d\n",
		stats.ActiveWorkers, stats.TotalProcessed, stats.FailedCount)
	assert(stats.TotalProcessed > 0, "Worker 池已处理任务")

	// 健康检查
	err = cli.HealthCheck()
	assert(err == nil, "HealthCheck 正常")

	// 动态调整 Worker
	cli.SetWorkers(8)
	time.Sleep(50 * time.Millisecond)
	stats = cli.Stats()
	assert(stats.ActiveWorkers >= 4, fmt.Sprintf("调整 Worker 后 ActiveWorkers=%d", stats.ActiveWorkers))

	// WithAsyncQueue 便捷选项
	cli2 := client.New(client.WithAsyncQueue(2, 64))
	cli2.Close()
	assert(true, "WithAsyncQueue 创建客户端成功")
}

// ============================================================================
// 3. Metrics 指标采集测试
// ============================================================================

func testMetrics() {
	section("3. Metrics 指标采集")

	// --- 标准 Metrics ---
	m := push.NewMetrics()

	m.Register("requests", 0, 1, 0, 10000)
	m.Register("errors", 0, 1, 0, 1000)
	m.Register("latency_ms", 0, 0.1, 0, 1e6)
	m.Register("cpu_percent", 0, 1, 0, 100)

	assert(m.Count() == 4, fmt.Sprintf("注册 4 个指标，实际 %d", m.Count()))
	keys := m.Keys()
	assert(len(keys) == 4, fmt.Sprintf("Keys 返回 %d 个名称", len(keys)))

	// Inc / Dec
	m.Inc("requests")
	m.Inc("requests")
	m.Inc("requests")
	assert(m.Get("requests") == 3, fmt.Sprintf("Inc 3 次后 requests=%.0f", m.Get("requests")))

	m.Dec("requests")
	assert(m.Get("requests") == 2, fmt.Sprintf("Dec 1 次后 requests=%.0f", m.Get("requests")))

	// IncBy
	m.IncBy("latency_ms", 42.5)
	m.IncBy("latency_ms", 7.5)
	assert(m.Get("latency_ms") == 50, fmt.Sprintf("IncBy 后 latency_ms=%.1f", m.Get("latency_ms")))

	// Set
	m.Set("cpu_percent", 72.3)
	assert(m.Get("cpu_percent") == 72.3, fmt.Sprintf("Set 后 cpu_percent=%.1f", m.Get("cpu_percent")))

	// GetDelta / Flush
	delta := m.GetDelta("requests")
	assert(delta == 2, fmt.Sprintf("GetDelta requests=%.0f (Flush 前)", delta))

	flushResult := m.Flush()
	assert(flushResult["requests"] == 2, fmt.Sprintf("Flush requests delta=%.0f", flushResult["requests"]))
	assert(m.GetDelta("requests") == 0, "Flush 后 GetDelta 为 0")

	// FlushAndReset
	m.Inc("requests")
	resetResult := m.FlushAndReset()
	assert(resetResult["requests"] == 3, fmt.Sprintf("FlushAndReset requests=%.0f", resetResult["requests"]))
	assert(m.Get("requests") == 0, "FlushAndReset 后值归零（initVal=0）")

	// Snapshot
	m.Inc("requests")
	snap := m.Snapshot()
	assert(snap["requests"] == 1, "Snapshot 正确")

	// Clear
	m.Clear()
	assert(m.Get("requests") == 0, "Clear 后归零")

	// 边界约束
	m.Register("bounded", 50, 10, 0, 100)
	for i := 0; i < 20; i++ {
		m.Inc("bounded")
	}
	assert(m.Get("bounded") == 100, fmt.Sprintf("上界截断 bounded=%.0f (max=100)", m.Get("bounded")))

	for i := 0; i < 20; i++ {
		m.Dec("bounded")
	}
	assert(m.Get("bounded") == 0, fmt.Sprintf("下界截断 bounded=%.0f (min=0)", m.Get("bounded")))

	// --- ShardedMetrics ---
	sm := push.NewShardedMetrics()
	sm.Register("sharded_counter", 0, 1, 0, 1e9)
	sm.Register("sharded_gauge", 0, 1, 0, 1e9)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				sm.Inc("sharded_counter")
			}
		}()
	}
	wg.Wait()
	assert(sm.Get("sharded_counter") == 100000, fmt.Sprintf("100 协程并发 Inc sharded_counter=%.0f", sm.Get("sharded_counter")))

	smKeys := sm.Keys()
	assert(len(smKeys) == 2, fmt.Sprintf("ShardedMetrics Keys 返回 %d 个", len(smKeys)))

	sm.Flush()
	sm.Clear()
	assert(sm.Get("sharded_counter") == 0, "ShardedMetrics Clear 后归零")
}

// ============================================================================
// 4. History 历史记录测试
// ============================================================================

func testHistory() {
	section("4. History 历史记录")

	type sample struct {
		ID    int
		Value string
	}

	// --- 标准 History ---
	h := push.NewHistory[sample](10)

	for i := 0; i < 15; i++ {
		h.Push("test", sample{ID: i, Value: fmt.Sprintf("val-%d", i)})
	}

	all := h.List("test", 0)
	assert(len(all) == 10, fmt.Sprintf("容量 10，写入 15 条后 List(0) 返回 %d 条", len(all)))
	assert(all[0].ID == 5 && all[9].ID == 14, fmt.Sprintf("覆盖后范围 [%d, %d]", all[0].ID, all[9].ID))

	last3 := h.List("test", 3)
	assert(len(last3) == 3, fmt.Sprintf("List(3) 返回 %d 条", len(last3)))
	assert(last3[2].ID == 14, fmt.Sprintf("最新一条 ID=%d", last3[2].ID))

	keys := h.Keys()
	assert(len(keys) >= 1, fmt.Sprintf("Keys 返回 %d 个 key", len(keys)))

	// 多 key 隔离
	h.Push("other", sample{ID: 100, Value: "other"})
	assert(len(h.List("other", 0)) == 1, "不同 key 隔离存储")

	// SetSizeByKey
	h.SetSizeByKey("test", 5)
	assert(len(h.List("test", 0)) <= 5, "SetSizeByKey 后截断到 5 条")
	assert(len(h.List("other", 0)) == 1, "其他 key 不受影响")

	// SetSize 全局
	h.SetSize(20)
	h.Push("test", sample{ID: 99, Value: "after-resize"})
	assert(len(h.List("test", 0)) >= 1, "SetSize 后可继续写入")

	// Clear
	h.Clear("test")
	assert(len(h.List("test", 0)) == 0, "Clear 后为空")

	// List 负数
	assert(h.List("test", -1) == nil, "List(-1) 返回 nil")

	// --- ShardedHistory ---
	sh := push.NewShardedHistory[sample](100)

	records := make([]sample, 50)
	for i := range records {
		records[i] = sample{ID: i, Value: fmt.Sprintf("sharded-%d", i)}
	}

	sh.BatchPush(func(s sample) string {
		if s.ID%2 == 0 {
			return "even"
		}
		return "odd"
	}, records)

	evenCount := sh.Count("even")
	oddCount := sh.Count("odd")
	totalAll := sh.CountAll()
	assert(evenCount == 25, fmt.Sprintf("even 分片 %d 条", evenCount))
	assert(oddCount == 25, fmt.Sprintf("odd 分片 %d 条", oddCount))
	assert(totalAll == 50, fmt.Sprintf("CountAll = %d", totalAll))

	// 单条 Push — 注意 Count 按 shard 统计，同一 shard 的所有 key 共享计数
	sh.ClearAll()
	sh.Push("custom", sample{ID: 999, Value: "custom"})
	assert(sh.Count("custom") >= 1, fmt.Sprintf("单条 Push 后 Count(custom)=%d", sh.Count("custom")))

	shList := sh.List("custom", 1)
	assert(len(shList) == 1, fmt.Sprintf("ShardedHistory List(custom, 1) 返回 %d 条", len(shList)))
	assert(shList[0].ID == 999, fmt.Sprintf("内容正确 ID=%d", shList[0].ID))

	// ClearAll
	sh.ClearAll()
	assert(sh.CountAll() == 0, "ClearAll 后全部清空")

	// SetSize
	sh.SetSize(50)
	assert(true, "SetSize 成功")
}

// ============================================================================
// 5. Timer 定时器测试
// ============================================================================

func testTimer() {
	section("5. Timer 定时器")

	var ticks atomic.Int32

	timer := push.NewTimer()
	assert(!timer.IsRunning(), "初始状态 IsRunning=false")

	// 启动
	timer.Start(50*time.Millisecond, func(t time.Time) {
		ticks.Add(1)
	})
	assert(timer.IsRunning(), "Start 后 IsRunning=true")

	// 等待几次 tick
	time.Sleep(300 * time.Millisecond)
	tickCount := ticks.Load()
	assert(tickCount >= 3, fmt.Sprintf("300ms 内触发 %d 次 tick (>=3)", tickCount))

	// 重复 Start 应无效果（CompareAndSwap 守护）
	ticksBefore := ticks.Load()
	timer.Start(50*time.Millisecond, func(t time.Time) {
		ticks.Add(100) // 如果真的重复启动，ticks 会暴涨
	})
	time.Sleep(150 * time.Millisecond)
	ticksAfter := ticks.Load()
	assert(ticksAfter < ticksBefore+100, "重复 Start 不创建新回调")

	// 停止
	beforeStop := ticks.Load()
	timer.Stop()
	assert(!timer.IsRunning(), "Stop 后 IsRunning=false")

	time.Sleep(200 * time.Millisecond)
	afterStop := ticks.Load()
	// 允许 Stop 时有一個在途 tick
	assert(afterStop <= beforeStop+2, fmt.Sprintf("Stop 后不再持续触发 (%d -> %d)", beforeStop, afterStop))

	// 多次 Stop 安全
	timer.Stop()
	timer.Stop()
	assert(true, "多次 Stop 不 panic")
}

// ============================================================================
// 6. 综合场景：定时采集 → 指标聚合 → 异步推送
// ============================================================================

func testIntegration(handler *mockHandler, serverURL string) {
	section("6. 综合场景：定时采集 → 推送")

	sm := push.NewShardedMetrics()
	sm.Register("cpu", 0, 1, 0, 100)
	sm.Register("memory", 0, 1, 0, 100)
	sm.Register("requests_total", 0, 1, 0, 1e9)

	history := push.NewShardedHistory[map[string]float64](200)

	p := push.New(
		push.WithBaseURL(serverURL),
		push.WithTimeout(3*time.Second),
	)

	var collectCount atomic.Int32
	timer := push.NewTimer()
	timer.Start(100*time.Millisecond, func(t time.Time) {
		cpu := float64(collectCount.Load()%80 + 10)
		mem := float64(collectCount.Load()%60 + 20)
		sm.Set("cpu", cpu)
		sm.Set("memory", mem)
		sm.IncBy("requests_total", float64(collectCount.Load()%5+1))

		delta := sm.Flush()
		history.Push("metrics", delta)

		data, _ := json.Marshal(delta)
		p.SendAsync(&push.Record{
			NeUID:      "ne-integration",
			RecordType: "integration_metrics",
			RecordData: data,
		}, nil)

		collectCount.Add(1)
	})

	time.Sleep(1 * time.Second)

	// 先停定时器，等回调完成后再关客户端
	timer.Stop()
	time.Sleep(200 * time.Millisecond)
	p.Close()

	total := collectCount.Load()
	fmt.Printf("  采集 %d 次，推送 %d 次\n", total, handler.typed("integration_metrics"))

	histItems := history.List("metrics", 5)
	assert(len(histItems) > 0, fmt.Sprintf("历史记录 %d 条 (查看最近 5 条)", len(histItems)))

	snap := sm.Snapshot()
	fmt.Println("  --- 最终指标快照 ---")
	for _, k := range []string{"cpu", "memory", "requests_total"} {
		fmt.Printf("  %-16s %.0f\n", k, snap[k])
	}
	assert(snap["requests_total"] > 0, "requests_total > 0")
}

// ============================================================================
// 7. 重试测试
// ============================================================================

func testRetry() {
	section("7. 重试机制")

	// 前 2 次失败、第 3 次成功
	var attempts atomic.Int32
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := attempts.Add(1)
		if a <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer failServer.Close()

	cli := client.New(
		client.WithRetry(3),
		client.WithTimeout(5*time.Second),
	)
	defer cli.Close()

	err := cli.Push(failServer.URL, map[string]any{"test": "retry"})
	assert(err == nil, fmt.Sprintf("重试成功（第 %d 次尝试）", attempts.Load()))
	assert(attempts.Load() == 3, fmt.Sprintf("共尝试 %d 次", attempts.Load()))

	// 超出重试次数应失败
	attempts.Store(0)
	failForever := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failForever.Close()

	err = cli.Push(failForever.URL, map[string]any{"test": "fail"})
	assert(err != nil, "超过重试次数返回错误")
}

// ============================================================================
// 8. 队列满降级测试
// ============================================================================

func testQueueFullDegradation() {
	section("8. 队列满降级")

	var received atomic.Int32
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	cli := client.New(
		client.WithWorkers(1),
		client.WithQueueSize(2),
		client.WithTimeout(10*time.Second),
	)
	defer cli.Close()

	for i := 0; i < 10; i++ {
		cli.AsyncPush(slowServer.URL, map[string]any{"index": i})
	}

	time.Sleep(2 * time.Second)
	count := received.Load()
	assert(count == 10, fmt.Sprintf("队列满降级后全部送达 (收到 %d/10)", count))
}

// ============================================================================
// main
// ============================================================================

func main() {
	fmt.Println("========================================")
	fmt.Println("  Push 模块全功能测试")
	fmt.Println("========================================")

	handler := newMockHandler()
	server := httptest.NewServer(handler)
	defer server.Close()
	serverURL := server.URL

	start := time.Now()

	testRecordPush(handler, serverURL)
	testClient(handler, serverURL)
	testMetrics()
	testHistory()
	testTimer()
	testIntegration(handler, serverURL)
	testRetry()
	testQueueFullDegradation()

	elapsed := time.Since(start)

	fmt.Println("\n========================================")
	fmt.Printf("  完成: %d 通过, %d 失败 (耗时 %v)\n", passed, failed, elapsed.Round(time.Millisecond))
	fmt.Println("========================================")

	if failed > 0 {
		log.Fatal("存在失败的测试用例")
	}
}
