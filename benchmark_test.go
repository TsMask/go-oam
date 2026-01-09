package oam

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

var (
	testServer *httptest.Server
	testURL    string
	testOnce   sync.Once
)

func initTestServer() {
	testOnce.Do(func() {
		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":0,"msg":"success"}`))
		}))
		testURL = testServer.URL
	})
}

func TestMain(m *testing.M) {
	initTestServer()
	code := m.Run()
	testServer.Close()
	os.Exit(code)
}

func TestCleanup(t *testing.T) {
	// This test is a placeholder to ensure TestMain's cleanup runs.
}

func BenchmarkAsyncPushLowConcurrency(b *testing.B) {
	initTestServer()
	fetch.AsyncInit(2, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			alarm := &model.Alarm{
				NeUid:       fmt.Sprintf("ne-%d", i),
				AlarmTime:   time.Now().UnixMilli(),
				AlarmId:     fmt.Sprintf("alarm-%d", i),
				AlarmCode:   1001,
				AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
				AlarmTitle:  "Test Alarm",
				AlarmStatus: model.ALARM_STATUS_ACTIVE,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			fetch.AsyncPush(ctx, testURL+"/push/alarm/receive", alarm)
			cancel()
			i++
		}
	})
}

func BenchmarkAsyncPushHighConcurrency(b *testing.B) {
	initTestServer()
	fetch.AsyncInit(2, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			alarm := &model.Alarm{
				NeUid:       fmt.Sprintf("ne-%d", i),
				AlarmTime:   time.Now().UnixMilli(),
				AlarmId:     fmt.Sprintf("alarm-%d", i),
				AlarmCode:   1001,
				AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
				AlarmTitle:  "Test Alarm",
				AlarmStatus: model.ALARM_STATUS_ACTIVE,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			fetch.AsyncPush(ctx, testURL+"/push/alarm/receive", alarm)
			cancel()
			i++
		}
	})
}

func BenchmarkAlarmServiceConcurrentPush(b *testing.B) {
	initTestServer()
	o := New(WithPush())
	alarm := &model.Alarm{
		NeUid:       "test-ne",
		AlarmTime:   time.Now().UnixMilli(),
		AlarmId:     "test-alarm",
		AlarmCode:   1001,
		AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
		AlarmTitle:  "Test Alarm",
		AlarmStatus: model.ALARM_STATUS_ACTIVE,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			alarm.AlarmId = fmt.Sprintf("alarm-%d", i)
			o.Push.AlarmURL(testURL+"/push/alarm/receive", alarm, 0)
			i++
		}
	})
}

func BenchmarkKPIServiceConcurrentOperations(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("kpi-key-%d", i%100)
			o.Push.KPIKeyInc(key)
			// o.Push.KPIKeyGet(key) // 暂时注释，避免影响基准测试结果
			i++
		}
	})
}

func BenchmarkCommonServiceConcurrentPush(b *testing.B) {
	initTestServer()
	o := New(WithPush())
	common := &model.Common{
		NeUid: "test-ne",
		Type:  "test-type",
		Data:  "test-data",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			common.Type = fmt.Sprintf("type-%d", i%10)
			o.Push.CommonURL(testURL+"/push/common/receive", common, 0)
			i++
		}
	})
}

func BenchmarkCommonServiceDifferentTypesConcurrentPush(b *testing.B) {
	initTestServer()
	o := New(WithPush())
	commonTypes := []string{"typeA", "typeB", "typeC", "typeD", "typeE"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			common := &model.Common{
				NeUid: "test-ne",
				Type:  commonTypes[i%len(commonTypes)],
				Data:  fmt.Sprintf("test-data-%d", i),
			}
			o.Push.CommonURL(testURL+"/push/common/receive", common, 0)
			i++
		}
	})
}

func BenchmarkNBStateServiceConcurrentPush(b *testing.B) {
	initTestServer()
	o := New(WithPush())
	nbStates := []string{model.NB_STATE_ON, model.NB_STATE_OFF}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			nbState := &model.NBState{
				NeUid:      "test-ne",
				RecordTime: time.Now().UnixMilli(),
				Address:    "test-address",
				DeviceName: "test-device",
				DeviceId:   int64(i),
				State:      nbStates[i%2],
				StateTime:  time.Now().UnixMilli(),
				Name:       "test-nb",
				Position:   "test-position",
			}
			o.Push.NBStateURL(testURL+"/push/nb_state/receive", nbState, 0)
			i++
		}
	})
}

func BenchmarkNBStateHistoryListConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())
	nbStates := []string{model.NB_STATE_ON, model.NB_STATE_OFF}

	for i := 0; i < 1000; i++ {
		nbState := &model.NBState{
			NeUid:      "test-ne",
			RecordTime: time.Now().UnixMilli(),
			Address:    "test-address",
			DeviceName: "test-device",
			DeviceId:   int64(i),
			State:      nbStates[i%2],
			StateTime:  time.Now().UnixMilli(),
			Name:       "test-nb",
			Position:   "test-position",
		}
		o.Push.NBStateURL(testURL+"/push/nb_state/receive", nbState, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.NBStateHistoryList(100)
		}
	})
}

func BenchmarkNBStateHistorySetSizeConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.NBStateHistorySetSize(1000)
			o.Push.NBStateHistorySetSize(2000)
		}
	})
}

func BenchmarkUENBServiceConcurrentPush(b *testing.B) {
	initTestServer()
	o := New(WithPush())
	uenbTypes := []string{model.UENB_TYPE_AUTH, model.UENB_TYPE_DETACH, model.UENB_TYPE_CM}
	uenbResults := []string{model.UENB_RESULT_AUTH_SUCCESS, model.UENB_RESULT_AUTH_NETWORK_FAILURE, model.UENB_RESULT_AUTH_INTERFACE_FAILURE, model.UENB_RESULT_AUTH_MAC_FAILURE, model.UENB_RESULT_AUTH_SYNC_FAILURE, model.UENB_RESULT_AUTH_NON_5G_AUTHENTICATION_NOT_ACCEPTED, model.UENB_RESULT_AUTH_RESPONSE_FAILURE, model.UENB_RESULT_AUTH_UNKNOWN}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			uenb := &model.UENB{
				NeUid:      "test-ne",
				RecordTime: time.Now().UnixMilli(),
				NBId:       "test-nb-id",
				CellId:     "test-cell-id",
				TAC:        "test-tac",
				IMSI:       fmt.Sprintf("test-imsi-%d", i),
				Type:       uenbTypes[i%len(uenbTypes)],
				Result:     uenbResults[i%len(uenbResults)],
			}
			o.Push.UENBURL(testURL+"/push/ue_nb/receive", uenb, 0)
			i++
		}
	})
}

func BenchmarkUENBHistoryListConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	for i := 0; i < 1000; i++ {
		uenb := &model.UENB{
			NeUid:  "test-ne",
			Type:   model.UENB_TYPE_AUTH,
			Result: model.UENB_RESULT_AUTH_SUCCESS,
		}
		o.Push.UENBURL(testURL+"/push/uenb/receive", uenb, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.UENBHistoryList(100)
		}
	})
}

func BenchmarkUENBHistorySetSizeConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.UENBHistorySetSize(1000)
			o.Push.UENBHistorySetSize(2000)
		}
	})
}

func BenchmarkCDRServiceConcurrentPush(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cdr := &model.CDR{
				NeUid:      "test-ne",
				RecordTime: time.Now().UnixMilli(),
				Data: map[string]interface{}{
					"sessionId": fmt.Sprintf("session-%d", i),
					"duration":  i * 10,
					"traffic":   i * 100,
					"IMSI":      fmt.Sprintf("imsi-%d", i),
					"IMEI":      fmt.Sprintf("imei-%d", i),
				},
			}
			o.Push.CDRURL(testURL+"/push/cdr/receive", cdr, 0)
			i++
		}
	})
}

func BenchmarkCDRHistoryListConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	for i := 0; i < 1000; i++ {
		cdr := &model.CDR{
			NeUid:      "test-ne",
			RecordTime: time.Now().UnixMilli(),
			Data: map[string]interface{}{
				"sessionId": fmt.Sprintf("session-%d", i),
				"duration":  i % 3600,
				"traffic":   float64(i%1024) / 1024,
				"IMSI":      fmt.Sprintf("imsi-%d", i),
			},
		}
		o.Push.CDRURL(testURL+"/push/cdr/receive", cdr, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			o.Push.CDRHistoryList(100)
			i++
		}
	})
}

func BenchmarkCDRHistorySetSizeConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.CDRHistorySetSize(1000)
			o.Push.CDRHistorySetSize(2000)
		}
	})
}

func BenchmarkHistoryListConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	for i := 0; i < 1000; i++ {
		alarm := &model.Alarm{
			NeUid:       "test-ne",
			AlarmTime:   time.Now().UnixMilli(),
			AlarmId:     fmt.Sprintf("alarm-%d", i),
			AlarmCode:   1001,
			AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
			AlarmTitle:  "Test Alarm",
			AlarmStatus: model.ALARM_STATUS_ACTIVE,
		}
		o.Push.AlarmURL(testURL+"/push/alarm/receive", alarm, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.AlarmHistoryList(100)
		}
	})
}

func BenchmarkHistorySetSizeConcurrent(b *testing.B) {
	initTestServer()
	o := New(WithPush())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.Push.AlarmHistorySetSize(1000)
			o.Push.AlarmHistorySetSize(2000)
		}
	})
}

func BenchmarkConfigConcurrentAccess(b *testing.B) {
	o := New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o.GetConfig().View(func(cfg *config.Config) {
				_ = cfg.NE.Type
			})
		}
	})
}

func TestStressTestHighQPS(t *testing.T) {
	initTestServer()
	fetch.AsyncInit(10, 1000)

	var successCount, failCount, fallbackCount atomic.Int64
	duration := 30 * time.Second
	workers := 200

	startTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Since(startTime) < duration {
				alarm := &model.Alarm{
					NeUid:       fmt.Sprintf("ne-%d", workerID),
					AlarmTime:   time.Now().UnixMilli(),
					AlarmId:     fmt.Sprintf("alarm-%d-%d", workerID, time.Now().UnixNano()),
					AlarmCode:   1001,
					AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
					AlarmTitle:  "Test Alarm",
					AlarmStatus: model.ALARM_STATUS_ACTIVE,
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := fetch.AsyncPush(ctx, testURL+"/push/alarm/receive", alarm)
				cancel()
				if err != nil {
					failCount.Add(1)
					if err.Error() == "push fallback error" {
						fallbackCount.Add(1)
					}
				} else {
					successCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalRequests := successCount.Load() + failCount.Load()
	qps := float64(totalRequests) / duration.Seconds()
	successRate := float64(successCount.Load()) / float64(totalRequests) * 100
	fallbackRate := float64(fallbackCount.Load()) / float64(totalRequests) * 100

	t.Logf("High Concurrency Stress Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Workers: %d", workers)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Failed: %d", failCount.Load())
	t.Logf("  Fallback: %d", fallbackCount.Load())
	t.Logf("  QPS: %.2f", qps)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Fallback Rate: %.2f%%", fallbackRate)

	if successRate < 95 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}
}

func TestMemoryLeakDetection(t *testing.T) {
	initTestServer()
	o := New(WithPush())
	fetch.AsyncInit(10, 1000)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m1, m2, m3 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 10000; i++ {
		alarm := &model.Alarm{
			NeUid:       "test-ne",
			AlarmTime:   time.Now().UnixMilli(),
			AlarmId:     fmt.Sprintf("alarm-%d", i),
			AlarmCode:   1001,
			AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
			AlarmTitle:  "Test Alarm",
			AlarmStatus: model.ALARM_STATUS_ACTIVE,
		}
		o.Push.AlarmURL(testURL+"/push/alarm/receive", alarm, 0)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&m2)

	for i := 0; i < 10000; i++ {
		alarm := &model.Alarm{
			NeUid:       "test-ne",
			AlarmTime:   time.Now().UnixMilli(),
			AlarmId:     fmt.Sprintf("alarm-%d", i+10000),
			AlarmCode:   1001,
			AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
			AlarmTitle:  "Test Alarm",
			AlarmStatus: model.ALARM_STATUS_ACTIVE,
		}
		o.Push.AlarmURL(testURL+"/push/alarm/receive", alarm, 0)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&m3)

	memIncrease1 := int64(0)
	if m2.Alloc > m1.Alloc {
		memIncrease1 = int64(m2.Alloc - m1.Alloc)
	}

	memIncrease2 := int64(0)
	if m3.Alloc > m2.Alloc {
		memIncrease2 = int64(m3.Alloc - m2.Alloc)
	}

	t.Logf("Memory Leak Detection Results:")
	t.Logf("  M1.Alloc: %d bytes", m1.Alloc)
	t.Logf("  M2.Alloc: %d bytes", m2.Alloc)
	t.Logf("  M3.Alloc: %d bytes", m3.Alloc)
	t.Logf("  Memory Increase (1st 10k): %d bytes", memIncrease1)
	t.Logf("  Memory Increase (2nd 10k): %d bytes", memIncrease2)
	t.Logf("  Memory Growth Rate: %.2f%%", float64(memIncrease2)/float64(memIncrease1)*100)

	if memIncrease2 > memIncrease1*2 {
		t.Errorf("Potential memory leak detected: memory growth rate too high")
	}
}

func TestQueueBackpressure(t *testing.T) {
	initTestServer()
	fetch.AsyncInit(4, 50)

	var blockedCount, successCount, fallbackCount atomic.Int64
	workers := 100
	requestsPerWorker := 200

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				alarm := &model.Alarm{
					NeUid:       fmt.Sprintf("ne-%d", workerID),
					AlarmTime:   time.Now().UnixMilli(),
					AlarmId:     fmt.Sprintf("alarm-%d-%d", workerID, j),
					AlarmCode:   1001,
					AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
					AlarmTitle:  "Test Alarm",
					AlarmStatus: model.ALARM_STATUS_ACTIVE,
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				start := time.Now()
				err := fetch.AsyncPush(ctx, testURL+"/push/alarm/receive", alarm)
				cancel()
				elapsed := time.Since(start)
				if err != nil {
					blockedCount.Add(1)
					if err.Error() == "push fallback error" {
						fallbackCount.Add(1)
					}
				} else {
					successCount.Add(1)
					if elapsed > 100*time.Millisecond {
						blockedCount.Add(1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalRequests := workers * requestsPerWorker
	qps := float64(totalRequests) / elapsed.Seconds()
	blockRate := float64(blockedCount.Load()) / float64(totalRequests) * 100
	fallbackRate := float64(fallbackCount.Load()) / float64(totalRequests) * 100

	t.Logf("Backpressure Handling Test Results:")
	t.Logf("  Queue Size: 50")
	t.Logf("  Workers: %d", workers)
	t.Logf("  Requests per Worker: %d", requestsPerWorker)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Blocked/Slow: %d", blockedCount.Load())
	t.Logf("  Fallback: %d", fallbackCount.Load())
	t.Logf("  Block Rate: %.2f%%", blockRate)
	t.Logf("  Fallback Rate: %.2f%%", fallbackRate)
	t.Logf("  QPS: %.2f", qps)
	t.Logf("  Elapsed Time: %v", elapsed)

	if blockRate > 30 {
		t.Errorf("Block rate too high: %.2f%% (indicates poor backpressure handling)", blockRate)
	}
}

func TestLongRunningStability(t *testing.T) {
	initTestServer()
	o := New(WithPush())
	fetch.AsyncInit(10, 1000)

	duration := 10 * time.Minute
	workers := 50

	var successCount, failCount atomic.Int64
	var errorCount atomic.Int64

	startTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Since(startTime) < duration {
				alarm := &model.Alarm{
					NeUid:       fmt.Sprintf("ne-%d", workerID),
					AlarmTime:   time.Now().UnixMilli(),
					AlarmId:     fmt.Sprintf("alarm-%d-%d", workerID, time.Now().UnixNano()),
					AlarmCode:   1001,
					AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
					AlarmTitle:  "Test Alarm",
					AlarmStatus: model.ALARM_STATUS_ACTIVE,
				}
				err := o.Push.AlarmURL(testURL+"/push/alarm/receive", alarm, 0)
				if err != nil {
					failCount.Add(1)
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}

				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	totalRequests := successCount.Load() + failCount.Load()
	qps := float64(totalRequests) / duration.Seconds()
	successRate := float64(successCount.Load()) / float64(totalRequests) * 100

	t.Logf("Long Running Stability Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Workers: %d", workers)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Failed: %d", failCount.Load())
	t.Logf("  QPS: %.2f", qps)
	t.Logf("  Success Rate: %.2f%%", successRate)

	if successRate < 99 {
		t.Errorf("Success rate degraded over time: %.2f%% (expected >= 99%%)", successRate)
	}
}

func TestConnectionPoolStress(t *testing.T) {
	initTestServer()
	fetch.AsyncInit(10, 1000)

	var successCount, failCount atomic.Int64
	duration := 20 * time.Second
	workers := 150

	startTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Since(startTime) < duration {
				alarm := &model.Alarm{
					NeUid:       fmt.Sprintf("ne-%d", workerID),
					AlarmTime:   time.Now().UnixMilli(),
					AlarmId:     fmt.Sprintf("alarm-%d-%d", workerID, time.Now().UnixNano()),
					AlarmCode:   1001,
					AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
					AlarmTitle:  "Test Alarm",
					AlarmStatus: model.ALARM_STATUS_ACTIVE,
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := fetch.AsyncPush(ctx, testURL+"/push/alarm/receive", alarm)
				cancel()
				if err != nil {
					failCount.Add(1)
				} else {
					successCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalRequests := successCount.Load() + failCount.Load()
	qps := float64(totalRequests) / duration.Seconds()
	successRate := float64(successCount.Load()) / float64(totalRequests) * 100

	t.Logf("Connection Pool Stress Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Workers: %d", workers)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Failed: %d", failCount.Load())
	t.Logf("  QPS: %.2f", qps)
	t.Logf("  Success Rate: %.2f%%", successRate)

	if successRate < 95 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}
}

func TestLockContention(t *testing.T) {
	initTestServer()
	o := New(WithPush())

	for i := 0; i < 5000; i++ {
		alarm := &model.Alarm{
			NeUid:       "test-ne",
			AlarmTime:   time.Now().UnixMilli(),
			AlarmId:     fmt.Sprintf("alarm-%d", i),
			AlarmCode:   1001,
			AlarmType:   model.ALARM_TYPE_COMMUNICATION_ALARM,
			AlarmTitle:  "Test Alarm",
			AlarmStatus: model.ALARM_STATUS_ACTIVE,
		}
		o.Push.AlarmURL(testURL+"/push/alarm/receive", alarm, 0)
	}

	workers := 100
	iterationsPerWorker := 1000

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterationsPerWorker; j++ {
				o.Push.AlarmHistoryList(100)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalOperations := workers * iterationsPerWorker
	opsPerSecond := float64(totalOperations) / elapsed.Seconds()

	t.Logf("Lock Contention Test Results:")
	t.Logf("  Workers: %d", workers)
	t.Logf("  Iterations per Worker: %d", iterationsPerWorker)
	t.Logf("  Total Operations: %d", totalOperations)
	t.Logf("  Elapsed Time: %v", elapsed)
	t.Logf("  Operations per Second: %.2f", opsPerSecond)

	if opsPerSecond < 10000 {
		t.Errorf("Lock contention too high: %.2f ops/sec (expected >= 10000)", opsPerSecond)
	}
}

func TestKPIConcurrentOperations(t *testing.T) {
	initTestServer()
	o := New(WithPush())

	workers := 50
	iterationsPerWorker := 2000

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterationsPerWorker; j++ {
				key := fmt.Sprintf("kpi-key-%d", j%100)
				o.Push.KPIKeyInc(key)
				o.Push.KPIKeyGet(key)
				o.Push.KPIKeyAdd(key, 1.5)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalOperations := workers * iterationsPerWorker * 3
	opsPerSecond := float64(totalOperations) / elapsed.Seconds()

	t.Logf("KPI Concurrent Operations Test Results:")
	t.Logf("  Workers: %d", workers)
	t.Logf("  Iterations per Worker: %d", iterationsPerWorker)
	t.Logf("  Total Operations: %d", totalOperations)
	t.Logf("  Elapsed Time: %v", elapsed)
	t.Logf("  Operations per Second: %.2f", opsPerSecond)

	if opsPerSecond < 50000 {
		t.Errorf("KPI operations too slow: %.2f ops/sec (expected >= 50000)", opsPerSecond)
	}
}
