// Package main provides usage examples and best practices for Push SDK.
package main

import (
	"fmt"
	"log"
	"time"

	push "github.com/tsmask/go-oam/push"
)

func ExampleBasicUsage() {
	fmt.Println("=== 1. Basic Usage ===")

	p := push.New(push.WithBaseURL("http://localhost:8080"))
	defer p.Close()

	m := push.NewMetrics()
	m.Register("cpu_usage", 0, 1, 0, 100)

	timer := push.NewTimer()
	timer.Start(5*time.Second, func(t time.Time) {
		m.IncBy("cpu_usage", 1.5)
		data := m.FlushAndReset()

		record := &push.Record{
			NeUID:      "ne-001",
			RecordType: "metrics",
			RecordData: data,
		}
		if err := p.Send(record); err != nil {
			log.Printf("Push failed: %v", err)
		}
	})

	time.Sleep(20 * time.Second)
	timer.Stop()
	fmt.Println("Basic usage example completed")
}

func ExampleAsyncPush() {
	fmt.Println("\n=== 2. Async Push ===")

	p := push.New(
		push.WithBaseURL("http://localhost:8080"),
		push.WithTimeout(30*time.Second),
	)
	defer p.Close()

	for i := 0; i < 100; i++ {
		record := &push.Record{
			NeUID:      "ne-001",
			RecordType: "alarm",
			RecordData: map[string]any{"level": i % 10},
		}

		if err := p.SendAsync(record); err != nil {
			log.Printf("Async push failed: %v", err)
		}
	}

	time.Sleep(time.Second)

	fmt.Println("Async push example completed")
}

func ExampleHighConcurrency() {
	fmt.Println("\n=== 3. High Concurrency ===")

	sm := push.NewShardedMetrics()
	sm.Register("counter", 0, 1, 0, 1e9)
	sm.Register("gauge", 50, 5, 0, 100)
	sm.Register("rate", 0, 0.1, 0, 1e6)

	done := make(chan bool, 100)
	start := time.Now()

	for i := 0; i < 100; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 1000; j++ {
				sm.IncBy("counter", 1)
				sm.Set("gauge", float64(id%100))
				sm.Get("rate")
			}
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	elapsed := time.Since(start)
	fmt.Printf("100k operations completed in %v\n", elapsed)
	fmt.Printf("Throughput: %.0f ops/s\n", 100000/elapsed.Seconds())

	snapshot := sm.Snapshot()
	fmt.Printf("Metrics snapshot: %+v\n", snapshot)

	p := push.New(push.WithBaseURL("http://localhost:8080"))
	defer p.Close()

	for i := 0; i < 1000; i++ {
		p.SendAsync(&push.Record{
			NeUID:      fmt.Sprintf("ne-%d", i%10),
			RecordType: "kpi",
			RecordData: map[string]any{"value": i},
		})
	}
	fmt.Printf("Batch insert 1000 records\n")
	fmt.Println("High concurrency example completed")
}

func ExamplePerformanceComparison() {
	fmt.Println("\n=== 4. Performance Comparison ===")

	m := push.NewMetrics()
	m.Register("standard_metric", 0, 1, 0, 1e9)

	sm := push.NewShardedMetrics()
	sm.Register("sharded_metric", 0, 1, 0, 1e9)

	standardStart := time.Now()
	for i := 0; i < 50000; i++ {
		m.Inc("standard_metric")
	}
	standardDuration := time.Since(standardStart)
	fmt.Printf("Standard Metrics: %v for 50k ops\n", standardDuration)

	shardedStart := time.Now()
	for i := 0; i < 50000; i++ {
		sm.Inc("sharded_metric")
	}
	shardedDuration := time.Since(shardedStart)
	fmt.Printf("Sharded Metrics: %v for 50k ops\n", shardedDuration)

	improvement := float64(standardDuration) / float64(shardedDuration)
	fmt.Printf("Sharded is %.2fx faster\n", improvement)
}

func ExampleTuning() {
	fmt.Println("\n=== 5. Performance Tuning ===")

	cli := push.NewClient()
	defer cli.Close()

	fmt.Println("Default config:")
	fmt.Printf("  Workers: %d\n", cli.Stats().ActiveWorkers)
	fmt.Printf("  QueueSize: %d\n", cli.Stats().QueueLength)

	if err := cli.HealthCheck(); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("Health check: OK")
	}

	stats := cli.Stats()
	fmt.Printf("Pool stats: ActiveWorkers=%d, QueueLength=%d\n",
		stats.ActiveWorkers, stats.QueueLength)

	fmt.Println("Performance tuning example completed")
}

func ExampleTroubleshooting() {
	fmt.Println("\n=== 6. Troubleshooting ===")

	fmt.Println("\n--- Issue: Connection Timeout ---")
	fmt.Println("Symptoms: Push requests hang or timeout")
	fmt.Println("Solutions:")
	fmt.Println("  1. Check network connectivity")
	fmt.Println("  2. Increase timeout: push.WithTimeout(60*time.Second)")
	fmt.Println("  3. Check server health")

	fmt.Println("\n--- Issue: Memory Growth ---")
	fmt.Println("Symptoms: Memory usage increases over time")
	fmt.Println("Solutions:")
	fmt.Println("  1. Use ShardedMetrics to reduce overhead")
	fmt.Println("  2. Clear unused metrics periodically")
	fmt.Println("  3. Use push.NewHistory() for fresh history")

	fmt.Println("\n--- Issue: High Latency ---")
	fmt.Println("Symptoms: Push latency > 100ms")
	fmt.Println("Solutions:")
	fmt.Println("  1. Use async push for non-critical data")
	fmt.Println("  2. Reduce retry attempts")
	fmt.Println("  3. Increase worker pool")

	fmt.Println("\n--- Issue: Queue Full ---")
	fmt.Println("Symptoms: 'async channel full' errors")
	fmt.Println("Solutions:")
	fmt.Println("  1. Use sync push as fallback")
	fmt.Println("  2. Add more workers")
	fmt.Println("  3. Use NewClient() for fresh queue")

	fmt.Println("\n--- Debugging Example ---")
	p := push.New(push.WithBaseURL("http://localhost:8080"))
	defer p.Close()

	cli := push.NewClient()
	defer cli.Close()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			stats := cli.Stats()
			if stats.QueueLength > 800 {
				log.Printf("WARNING: Queue nearly full: %d", stats.QueueLength)
			}
			if stats.FailedCount > 0 {
				log.Printf("WARNING: Failed pushes: %d", stats.FailedCount)
			}
		}
	}()

	time.Sleep(5 * time.Second)
	fmt.Println("Debugging example completed")
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  Push SDK Usage Guide")
	fmt.Println("========================================")

	ExampleBasicUsage()
	ExampleAsyncPush()
	ExampleHighConcurrency()
	ExamplePerformanceComparison()
	ExampleTuning()
	ExampleTroubleshooting()

	fmt.Println("\n========================================")
	fmt.Println("  All examples completed")
	fmt.Println("========================================")
}
