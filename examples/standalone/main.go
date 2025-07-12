package main

import (
	"fmt"
	"time"

	"github.com/tsmask/go-oam"

	"github.com/gin-gonic/gin"
)

// 独立运行
func main() {
	o := oam.New(&oam.Opts{
		Dev:      true,
		ConfPath: "./local/oam.yaml",
		License: &oam.License{
			NeType:     "NE",
			Version:    "1.0",
			SerialNum:  "1234567890",
			ExpiryDate: "2025-12-31",
			Capability: 100,
		},
	})

	// 周期模拟
	duration := 5 * time.Second
	timer := time.NewTimer(duration)
	go func() {
		for {
			t := <-timer.C

			// 发告警
			alarmId := fmt.Sprintf("100_%d", t.UnixMilli())
			alarm := oam.Alarm{
				NeUid:             "neUid",                            // 网元唯一标识
				AlarmId:           alarmId,                            // 告警ID
				AlarmCode:         100,                                // 告警状态码
				AlarmType:         oam.ALARM_TYPE_COMMUNICATION_ALARM, // 告警类型 CommunicationAlarm,EquipmentAlarm,ProcessingFailure,EnvironmentalAlarm,QualityOfServiceAlarm
				AlarmTitle:        "Alarm Test",                       // 告警标题
				PerceivedSeverity: oam.ALARM_SEVERITY_MAJOR,           // 告警级别 Critical,Major,Minor,Warning,Event
				AlarmStatus:       oam.ALARM_STATUS_ACTIVE,            // 告警状态 Clear,Active
				SpecificProblem:   "Alarm Test",                       // 告警问题原因
				SpecificProblemID: "100",                              // 告警问题原因ID
				AddInfo:           "addInfo",                          // 告警辅助信息
				LocationInfo:      "locationInfo",                     // 告警定位信息
			}
			errs := oam.AlarmPush("http", "192.168.5.58:29565", &alarm)
			if errs != nil {
				fmt.Println("==> Send err Alarm:", errs.Error())
			} else {
				fmt.Println("==> Send ok AlarmSeq:", alarm.AlarmId)
			}

			// 发终端接入基站
			uenb := oam.UENB{
				NeUid:  "neUid",                      // 网元唯一标识
				NBId:   fmt.Sprint(t.Second()),       // 基站ID
				CellId: "1",                          // 小区ID
				TAC:    "4388",                       // TAC
				IMSI:   "460991100000000",            // IMSI
				Result: oam.UENB_RESULT_AUTH_SUCCESS, // 结果值
				Type:   oam.UENB_TYPE_AUTH,           // 终端接入基站类型
			}
			errs = oam.UENBPush("http", "192.168.5.58:29565", &uenb)
			if errs != nil {
				fmt.Println("==> Send err UENBTime:", errs.Error())
			} else {
				fmt.Println("==> Send ok UENBTime:", uenb.RecordTime)
			}

			// 发话单
			cdr := oam.CDR{
				NeUid: "neUid", // 网元唯一标识
				Data: map[string]any{
					"seqNumber":    true,
					"callDuration": t.Second(),
					"recordType":   "MT",
					"cause":        200,
					"releaseTime":  t.UnixMilli(),
				},
			}
			errs = oam.CDRPush("http", "192.168.5.58:29565", &cdr)
			if errs != nil {
				fmt.Println("==> Send err CDR:", errs.Error())
			} else {
				fmt.Println("==> Send ok CDR:", cdr.RecordTime)
			}

			// 发指标
			oam.KPIKeyInc("Test.A.01")
			oam.KPIKeyInc("Test.A.02")
			oam.KPIKeySet("Test.A.03", float64(t.Second()))

			// 重置定时器，按指定周期执行
			timer.Reset(duration)
		}
	}()

	// 告警历史清除
	oam.AlarmHistoryClearTimer()
	// UENB 终端接入基站历史清除
	oam.UENBHistoryClearTimer()
	// 话单历史清除
	oam.CDRHistoryClearTimer()

	o.RouteAdd(func(r gin.IRouter) {
		// 网管接收端收告警
		oam.AlarmReceiveRoute(r, func(alarm oam.Alarm) error {
			fmt.Println("<== Receive Alarm", alarm)
			return nil
		})
		// 网管接收端收终端接入基站
		oam.UENBReceiveRoute(r, func(uenb oam.UENB) error {
			fmt.Println("<== Receive UENB", uenb)
			return nil
		})
		// 网管接收端收话单
		oam.CDRReceiveRoute(r, func(cdr oam.CDR) error {
			fmt.Println("<== Receive CDR", cdr)
			return nil
		})
		// 指标发送测试
		oam.KPITimerStart("http", "192.168.5.58:29565", "neUid", 10*time.Second)
		// 网管接收端收KPI
		oam.KPIReceiveRoute(r, func(kpi oam.KPI) error {
			fmt.Println("<== Receive KPI", kpi)
			return nil
		})
	})

	if err := o.Run(); err != nil {
		fmt.Printf("oam run fail: %s\n", err.Error())
	}
}
