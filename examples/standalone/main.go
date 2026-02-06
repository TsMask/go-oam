package main

import (
	"fmt"
	"time"

	"github.com/tsmask/go-oam"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/modules/common"
	"github.com/tsmask/go-oam/modules/state"

	"github.com/gin-gonic/gin"
)

// 独立运行
func main() {
	o := oam.New(
		oam.WithNEConfig(config.NEConfig{
			Type:       "NE",
			Version:    "1.0",
			SerialNum:  "1234567890",
			ExpiryDate: "2025-12-31",
			NbNumber:   10,
			UeNumber:   100,
		}),
		oam.WithOMCConfig(config.OMCConfig{
			URL:     "http://127.0.0.1:29565",
			NeUID:   "1234567890",
			CoreUID: "1234567890",
		}),
		oam.WithPush(), // 开启推送功能
	)

	// 周期模拟
	duration := 5 * time.Second
	timer := time.NewTimer(duration)
	go func() {
		for {
			t := <-timer.C

			// 发通用记录
			common := oam.Common{
				Type: "common",
				Data: map[string]any{
					"commonSecond": t.Second(),
					"commonTime":   t.UnixMilli(),
				},
			}
			commonErr := o.Push.Common(&common)
			if commonErr != nil {
				fmt.Println("==> Send err Common:", commonErr.Error())
			} else {
				fmt.Println("==> Send ok Common:", common.RecordTime)
			}

			// 发告警
			alarmId := fmt.Sprintf("100_%d", t.UnixMilli())
			alarm := oam.Alarm{
				AlarmId:           alarmId,                      // 告警ID
				AlarmCode:         100,                          // 告警状态码
				AlarmType:         oam.ALARM_TYPE_COMMUNICATION, // 告警类型
				AlarmTitle:        "Alarm Test",                 // 告警标题
				PerceivedSeverity: oam.ALARM_SEVERITY_MAJOR,     // 告警级别
				AlarmStatus:       oam.ALARM_STATUS_ACTIVE,      // 告警状态
				SpecificProblem:   "Alarm Test",                 // 告警问题原因
				SpecificProblemID: "100",                        // 告警问题原因ID
				AddInfo:           "addInfo",                    // 告警辅助信息
				LocationInfo:      "locationInfo",               // 告警定位信息
			}
			errs := o.Push.Alarm(&alarm)
			if errs != nil {
				fmt.Println("==> Send err Alarm:", errs.Error())
			} else {
				fmt.Println("==> Send ok AlarmSeq:", alarm.AlarmId)
			}

			// 发终端接入基站
			uenb := oam.UENB{
				NBId:   fmt.Sprint(t.Second()),       // 基站ID
				NBIp:   "127.0.0.1",                  // 基站IP
				CellId: "1",                          // 小区ID
				TAC:    "4388",                       // TAC
				IMSI:   "460991100000000",            // IMSI
				MSISDN: "8613800138000",              // MSISDN
				Result: oam.UENB_RESULT_AUTH_SUCCESS, // 结果值
				Type:   oam.UENB_TYPE_AUTH,           // 终端接入基站类型
			}
			errs = o.Push.UENB(&uenb)
			if errs != nil {
				fmt.Println("==> Send err UENBTime:", errs.Error())
			} else {
				fmt.Println("==> Send ok UENBTime:", uenb.RecordTime)
			}

			// 发基站状态
			nbState := oam.NBState{
				Address:    "192.168.101.112",      // 基站地址
				DeviceName: "TestNB",               // 基站设备名称
				State:      oam.NB_STATE_OFF,       // 基站状态 ON/OFF
				StateTime:  time.Now().UnixMilli(), // 基站状态时间
				Name:       "TestName",             // 基站名称 网元标记
				Position:   "TestPosition",         // 基站位置 网元标记
			}
			errs = o.Push.NBState(&nbState)
			if errs != nil {
				fmt.Println("==> Send err NBStateTime:", errs.Error())
			} else {
				fmt.Println("==> Send ok NBStateTime:", nbState.RecordTime)
			}

			// 发话单
			cdr := oam.CDR{
				Data: map[string]any{
					"cdrSecond": t.Second(),
					"cdrTime":   t.UnixMilli(),
				},
			}
			errs = o.Push.CDR(&cdr)
			if errs != nil {
				fmt.Println("==> Send err CDRTime:", errs.Error())
			} else {
				fmt.Println("==> Send ok CDRTime:", cdr.RecordTime)
			}

			// 发KPI
			o.Push.KPIKeyInc("test_inc")
			o.Push.KPIKeySet("test_set", 100.1)
			errs = o.Push.KPI()
			if errs != nil {
				fmt.Println("==> Send err KPI:", errs.Error())
			} else {
				fmt.Println("==> Send ok KPI")
			}

			timer.Reset(duration)
		}
	}()

	o.SetupRoute(func(r gin.IRouter) {
		common.SetupRoute(r)
		state.SetupRoute(r)
		// o.Push 中的所有 Service 实例
		o.SetupPushRoute(r)
	})

	// 运行 SDK 逻辑
	if err := o.Run(); err != nil {
		fmt.Printf("OAM SDK run error: %v\n", err)
	}
}
