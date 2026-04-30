package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tsmask/go-oam/push"
)

func periodic() {
	p := push.New(push.WithBaseURL("http://192.168.7.7:8000"))

	mCPU := push.NewMetrics()
	mCPU.Register("count", 0, 1, 0, 10000)
	mCPU.Register("error_count", 0, 1, 0, 1000)

	mMem := push.NewMetrics()
	mMem.Register("usage", 50, 5, 0, 100)

	t1 := push.NewTimer()
	t2 := push.NewTimer()

	t1.Start(3*time.Second, func(t time.Time) {
		data := mCPU.FlushAndReset()
		log.Printf("%s CPU stats: %v", t.Format("15:04:05"), data)

		record := &push.Record{
			NeUID:      "ne-001",
			RecordType: "kpi",
			RecordData: data,
		}
		if err := p.Send(record); err != nil {
			log.Printf("Send failed: %v", err)
		}
	})

	t2.Start(10*time.Second, func(t time.Time) {
		usage := mMem.Get("usage")
		if usage > 80 {
			data := map[string]float64{"usage": usage}
			record := &push.Record{
				NeUID:      "ne-001",
				RecordType: "alarm",
				RecordData: data,
			}
			p.Send(record)
		}
	})

	log.Printf("Timers started, press Ctrl+C to stop")
	time.Sleep(20 * time.Second)

	log.Printf("Stopping timers...")
	t1.Stop()
	t2.Stop()

	p.Close()
	log.Printf("Done")
}

func uni() {
	baseURL := "http://192.168.7.7:8000"
	neUid := "abcd1234"

	p := push.New(
		push.WithBaseURL(baseURL),
		push.WithTimeout(30*time.Second),
		push.WithRetry(3),
	)

	log.Printf("Push 服务已启动，推送地址: %s\n", baseURL)

	alarmRecord := &push.Record{
		NeUID:      neUid,
		RecordTime: time.Now().UnixMilli(),
		RecordType: "alarm",
		RecordData: map[string]any{
			"title": "Error IP",
		},
	}

	log.Printf("发送告警推送...")
	if err := p.Send(alarmRecord); err != nil {
		log.Printf("推送失败: %v", err)
	} else {
		log.Printf("推送成功")
	}

	cdrRecord := &push.Record{
		NeUID:      neUid,
		RecordTime: time.Now().UnixMilli(),
		RecordType: "cdr",
		RecordData: map[string]any{
			"imsi": "1234",
		},
	}

	log.Printf("发送 CDR 推送...")
	if err := p.SendAsync(cdrRecord); err != nil {
		log.Printf("推送失败: %v", err)
	} else {
		log.Printf("推送成功")
	}

	log.Printf("发送异步告警推送...")
	if err := p.SendAsyncURL(baseURL+"/api/alarm", alarmRecord); err != nil {
		log.Printf("异步推送失败: %v", err)
	} else {
		log.Printf("异步推送成功")
	}

	time.Sleep(500 * time.Millisecond)

	p.Close()
	log.Printf("Push 服务已关闭")
}

func main() {
	periodic()
	fmt.Println("======= Next =====")
	uni()
}
