package push

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/pkg/fetch"
	"github.com/tsmask/go-oam/pkg/ringbuffer"
)

// KPI_PUSH_URI 指标推送URI地址 POST
const KPI_PUSH_URI = "/push/kpi/receive"

// float64互转int64 精度控制，支持3位小数精度
const precisionMultiplier = 1000

// KPI 指标信息对象
type KPI struct {
	NeUid       string             `json:"neUid" binding:"required"`       // 网元唯一标识
	RecordTime  int64              `json:"recordTime" binding:"required"`  // 记录时间 时间戳毫秒，Push时自动填充
	Granularity int64              `json:"granularity" binding:"required"` // 时间间隔 5/10/.../60/300 (秒)
	Data        map[string]float64 `json:"data"  binding:"required"`       // 指标信息
}

// KPIService 指标服务
type KPIService struct {
	NeUid              string                      // 网元唯一标识
	Granularity        time.Duration               // 指标缓存时间粒度
	data               sync.Map                    // 存储string -> *atomic.Int64
	kpiTimerCancel     context.CancelFunc           // KPI 定时发送取消函数
	kpiHistorys        *ringbuffer.RingBuffer[KPI]  // KPI历史记录（环形缓冲区）
	kpiHistorysMaxSize atomic.Int32                 // 最大历史记录数量
}

// NewKPIService 创建KPI服务
func NewKPIService(neUid string, granularity time.Duration) *KPIService {
	k := &KPIService{
		NeUid:       neUid,
		Granularity: granularity,
		kpiHistorys: ringbuffer.NewRingBuffer[KPI](4096),
	}
	k.kpiHistorysMaxSize.Store(4096)
	return k
}

// KPITimerStart KPI定时发送
func (s *KPIService) KPITimerStart(urlGetter func() string) {
	if s == nil {
		return
	}
	s.KPITimerStop()
	ctx, cancel := context.WithCancel(context.Background())
	s.kpiTimerCancel = cancel
	kpiTimer := time.NewTimer(s.Granularity)

	go func() {
		defer kpiTimer.Stop()
		fail := 0
		for {
			select {
			case <-kpiTimer.C:
				dataMap := s.safeGetAllData()
				if len(dataMap) != 0 {
					granularity := int64(s.Granularity.Seconds())
					url := urlGetter()
					if url != "" {
						err := s.Send(url, s.NeUid, granularity, dataMap, 0)
						if err != nil {
							log.Printf("[OAM] kpi timer send failed NeUid: %s, Granularity: %ds\n%s\n", s.NeUid, granularity, err.Error())
							fail++
						} else {
							fail = 0
							s.safeClearData()
						}
					}
				}
				delay := s.Granularity
				if fail == 1 {
					delay = s.Granularity * 2
				} else if fail >= 2 {
					delay = s.Granularity * 4
				}
				kpiTimer.Reset(delay)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// KPITimerStop 停止KPI定时发送
func (s *KPIService) KPITimerStop() {
	if s == nil {
		return
	}
	if s.kpiTimerCancel != nil {
		s.kpiTimerCancel()
		s.kpiTimerCancel = nil
	}
}

func (s *KPIService) getOrCreateAtomicValue(key string) *atomic.Int64 {
	if val, ok := s.data.Load(key); ok {
		return val.(*atomic.Int64)
	}
	newVal := &atomic.Int64{}
	actual, _ := s.data.LoadOrStore(key, newVal)
	return actual.(*atomic.Int64)
}

// KeySet 对Key原子设置
func (s *KPIService) KeySet(key string, v float64) {
	if s == nil {
		return
	}
	s.getOrCreateAtomicValue(key).Store(int64(v * precisionMultiplier))
}

// KeyGet 对Key原子获取
func (s *KPIService) KeyGet(key string) float64 {
	if s == nil {
		return 0
	}
	val, ok := s.data.Load(key)
	if !ok {
		return 0
	}
	return float64(val.(*atomic.Int64).Load()) / precisionMultiplier
}

// KeyInc 对Key原子累加1
func (s *KPIService) KeyInc(key string) {
	if s == nil {
		return
	}
	s.getOrCreateAtomicValue(key).Add(precisionMultiplier)
}

// KeyDec 对Key原子累减1
func (s *KPIService) KeyDec(key string) {
	if s == nil {
		return
	}
	s.getOrCreateAtomicValue(key).Add(-precisionMultiplier)
}

// KeyAdd 原子增加指定值
func (s *KPIService) KeyAdd(key string, v float64) {
	if s == nil {
		return
	}
	s.getOrCreateAtomicValue(key).Add(int64(v * precisionMultiplier))
}

// KeyDel 删除指定的键
func (s *KPIService) KeyDel(key string) {
	if s == nil {
		return
	}
	s.data.Delete(key)
}

func (s *KPIService) safeGetAllData() map[string]float64 {
	dataMap := make(map[string]float64)
	s.data.Range(func(key, value any) bool {
		k := key.(string)
		dataMap[k] = float64(value.(*atomic.Int64).Load()) / precisionMultiplier
		return true
	})
	return dataMap
}

func (s *KPIService) safeClearData() {
	s.data.Range(func(key, _ any) bool {
		s.data.Delete(key)
		return true
	})
}

// HistoryList 线程安全地获取历史列表
func (s *KPIService) HistoryList(n int) []KPI {
	if s == nil {
		return []KPI{}
	}
	if n < 0 {
		return []KPI{}
	}
	if n == 0 {
		return s.kpiHistorys.GetAll()
	}
	return s.kpiHistorys.GetLast(n)
}

// HistorySetSize 安全地修改最大历史记录数量
func (s *KPIService) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}
	s.kpiHistorysMaxSize.Store(int32(newSize))
	s.kpiHistorys.Resize(newSize)
}

// PushURL 推送KPI到指定URL
func (s *KPIService) PushURL(url string, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	dataMap := s.safeGetAllData()
	if len(dataMap) == 0 {
		return nil
	}
	granularity := int64(s.Granularity.Seconds())
	err := s.Send(url, s.NeUid, granularity, dataMap, timeout)
	if err == nil {
		s.safeClearData()
	}
	return err
}

// Send 发送KPI数据
func (s *KPIService) Send(url, neUid string, granularity int64, dataMap map[string]float64, timeout time.Duration) error {
	if s == nil {
		return nil
	}
	k := KPI{
		Data:        dataMap,
		Granularity: granularity,
		RecordTime:  time.Now().UnixMilli(),
		NeUid:       neUid,
	}
	s.kpiHistorys.Push(k)
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetch.AsyncPush(ctx, url, k)
}
