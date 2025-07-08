package service

import (
	"context"
	"sync"
	"time"

	"github.com/tsmask/go-oam/src/framework/fetch"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/modules/push/model"
)

// KPI_PUSH_URI 指标推送URI地址 POST
const KPI_PUSH_URI = "/push/kpi/receive"

// kpiHistorys KPI历史 每小时重新记录，保留一小时
var kpiHistorys []model.KPI = make([]model.KPI, 0)

// KPI 指标服务
type KPI struct {
	NeUid          string             // 网元唯一标识
	Granularity    time.Duration      // 指标缓存时间粒度
	data           sync.Map           // 指标缓存
	kpiTimerCancel context.CancelFunc // KPI 定时发送取消函数
}

// kpiClearDuration 计算到下一个时间间隔
func kpiClearDuration() time.Duration {
	now := time.Now()
	// 下一个整点
	nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	return nextHour.Sub(now)
}

// KPITimerStart KPI定时发送
// duration 为发送周期，单位为 time.Duration
func (s *KPI) KPITimerStart(url string) {
	// 先关闭当前定时器
	s.KPITimerStop()

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	s.kpiTimerCancel = cancel

	// 创建 KPI 发送定时器，在指定的周期后触发第一次执行
	kpiTimer := time.NewTimer(s.Granularity)
	// 计算到下一个整点的时间间隔，创建历史记录清空定时器
	historyClearTimer := time.NewTimer(kpiClearDuration())

	go func() {
		defer kpiTimer.Stop()
		defer historyClearTimer.Stop()
		for {
			select {
			case <-kpiTimer.C:
				// 将 sync.Map 转换为 map[string]float64
				dataMap := make(map[string]float64)
				s.data.Range(func(key, value any) bool {
					dataMap[key.(string)] = value.(float64)
					return true
				})
				// 执行 KPI 发送操作
				k := model.KPI{
					NeUid:       s.NeUid, // 网元唯一标识
					Granularity: int64(s.Granularity.Seconds()),
					RecordTime:  time.Now().UnixMilli(),
					Data:        dataMap,
				}

				// 发送
				_, err := fetch.PostJSON(url, k, nil)
				if err != nil {
					logger.Errorf("KPITimer PostJSON error: %v", err)
				}
				// 记录历史
				kpiHistorys = append(kpiHistorys, k)
				// 清空 sync.Map
				s.data = sync.Map{}
				// 重置 KPI 定时器，按指定周期执行
				kpiTimer.Reset(s.Granularity)
			case now := <-historyClearTimer.C:
				// 计算小时前的起始时间戳
				hourAgo := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 0, 0, 0, now.Location()).UnixMilli()

				// 过滤出最近小时的 KPI 记录
				var recentHistory []model.KPI
				for _, kpi := range kpiHistorys {
					if kpi.RecordTime >= hourAgo {
						recentHistory = append(recentHistory, kpi)
					}
				}
				// 更新 kpiHistorys 为最近小时的记录
				kpiHistorys = recentHistory

				// 重置历史记录清空定时器，准备下一个整点
				historyClearTimer.Reset(kpiClearDuration())
			case <-ctx.Done():
				return
			}
		}
	}()
}

// KPITimerStop 停止KPI定时发送
func (s *KPI) KPITimerStop() {
	if s.kpiTimerCancel != nil {
		s.kpiTimerCancel()
		s.kpiTimerCancel = nil
	}
}

// KeySet 对Key原子设置
func (s *KPI) KeySet(key string, v float64) {
	s.data.Store(key, v)
}

// KeyInc 对Key原子累加
func (s *KPI) KeyInc(key string) {
	s.data.Store(key, s.KeyGet(key)+1)
}

// KeyDec 对Key原子累减
func (s *KPI) KeyDec(key string) {
	s.data.Store(key, s.KeyGet(key)-1)
}

// KeyGet 对Key原子获取
func (s *KPI) KeyGet(key string) float64 {
	value, ok := s.data.Load(key)
	if !ok {
		return 0
	}
	return value.(float64)
}

// KPIHistoryList KPI历史列表
func KPIHistoryList() []model.KPI {
	return kpiHistorys
}

// KPISend 发送KPI
func KPISend(url, neUid string, granularity int64, dataMap map[string]float64) error {
	// 执行 KPI 发送操作
	k := model.KPI{
		Data:        dataMap,
		Granularity: granularity,
		RecordTime:  time.Now().UnixMilli(),
		NeUid:       neUid,
	}

	// 发送
	_, err := fetch.PostJSON(url, k, nil)
	if err != nil {
		logger.Errorf("KPISend PostJSON error: %v", err)
		return err
	}
	// 记录历史
	kpiHistorys = append(kpiHistorys, k)
	return nil
}
