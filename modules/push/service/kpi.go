package service

import (
	"context"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tsmask/go-oam/framework/fetch"
	"github.com/tsmask/go-oam/modules/push/model"
)

// KPI_PUSH_URI 指标推送URI地址 POST
const KPI_PUSH_URI = "/push/kpi/receive"

// float64互转uint64 精度控制，支持3位小数精度
const precisionMultiplier = 1000

var (
	kpiHistorys           []model.KPI  // KPI历史记录
	kpiHistorysMux        sync.RWMutex // 保护kpiHistorys的并发访问
	kpiHistorysMaxSize    = 4096       // 最大历史记录数量
	kpiHistorysMaxSizeMux sync.RWMutex // 保护修改数量的并发访问
)

// KPI 指标服务
type KPI struct {
	NeUid          string             // 网元唯一标识
	Granularity    time.Duration      // 指标缓存时间粒度
	data           sync.Map           // 存储string -> *atomic.Uint64
	clearMutex     sync.Mutex         // 保护清空操作
	kpiTimerCancel context.CancelFunc // KPI 定时发送取消函数

}

// KPITimerStart KPI定时发送
func (s *KPI) KPITimerStart(url string) {
	// 先关闭当前定时器
	s.KPITimerStop()

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	s.kpiTimerCancel = cancel

	// 创建 KPI 发送定时器
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
					err := KPISend(url, s.NeUid, granularity, dataMap)
					if err != nil {
						log.Printf("[OAM] kpi timer send failed NeUid: %s, Granularity: %ds\n%s\n", s.NeUid, granularity, err.Error())
						fail++
					} else {
						fail = 0
						s.safeClearData()
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
func (s *KPI) KPITimerStop() {
	if s.kpiTimerCancel != nil {
		s.kpiTimerCancel()
		s.kpiTimerCancel = nil
	}
}

// getOrCreateAtomicValue 获取或创建atomic值
func (s *KPI) getOrCreateAtomicValue(key string) *atomic.Uint64 {
	// 快速路径：尝试加载已存在的值
	if val, ok := s.data.Load(key); ok {
		return val.(*atomic.Uint64)
	}

	// 慢速路径：创建新的atomic值
	newVal := &atomic.Uint64{}

	// 使用LoadOrStore确保线程安全，防止重复创建
	actual, _ := s.data.LoadOrStore(key, newVal)
	return actual.(*atomic.Uint64)
}

// KeySet 对Key原子设置
func (s *KPI) KeySet(key string, v float64) {
	if s == nil {
		return
	}

	// 将float64转换为uint64，使用放大因子保持精度
	uintVal := uint64(v * precisionMultiplier)
	atomicVal := s.getOrCreateAtomicValue(key)
	atomicVal.Store(uintVal)
}

// KeyGet 对Key原子获取
func (s *KPI) KeyGet(key string) float64 {
	if s == nil {
		return 0
	}

	val, ok := s.data.Load(key)
	if !ok {
		return 0
	}

	atomicVal := val.(*atomic.Uint64)
	// 将uint64转换回float64
	return float64(atomicVal.Load()) / precisionMultiplier
}

// KeyInc 对Key原子累加1
func (s *KPI) KeyInc(key string) {
	if s == nil {
		return
	}

	atomicVal := s.getOrCreateAtomicValue(key)
	atomicVal.Add(precisionMultiplier)
}

// KeyDec 对Key原子累减1
func (s *KPI) KeyDec(key string) {
	if s == nil {
		return
	}

	atomicVal := s.getOrCreateAtomicValue(key)
	atomicVal.Add(^uint64(precisionMultiplier - 1))
}

// KeyAdd 原子增加指定值
func (s *KPI) KeyAdd(key string, delta float64) {
	if s == nil {
		return
	}

	atomicVal := s.getOrCreateAtomicValue(key)
	deltaUint := uint64(math.Abs(delta * precisionMultiplier))

	if delta >= 0 {
		atomicVal.Add(deltaUint)
	} else {
		atomicVal.Add(^uint64(deltaUint - 1))
	}
}

// KeyDel 删除指定的键
func (s *KPI) KeyDel(key string) {
	if s == nil {
		return
	}
	s.data.Delete(key)
}

// safeGetAllData 线程安全地获取所有数据
func (s *KPI) safeGetAllData() map[string]float64 {
	dataMap := make(map[string]float64)
	s.data.Range(func(key, value any) bool {
		k := key.(string)
		atomicVal := value.(*atomic.Uint64)
		dataMap[k] = float64(atomicVal.Load()) / precisionMultiplier
		return true
	})
	return dataMap
}

// safeClearData 线程安全地清空数据
func (s *KPI) safeClearData() {
	s.clearMutex.Lock()
	defer s.clearMutex.Unlock()

	// 遍历并删除所有键，避免赋值sync.Map
	s.data.Range(func(key, _ any) bool {
		s.data.Delete(key)
		return true
	})
}

// KPIHistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func KPIHistoryList(n int) []model.KPI {
	kpiHistorysMux.RLock()
	defer kpiHistorysMux.RUnlock()

	if n < 0 {
		return []model.KPI{}
	}

	// 计算要返回的记录数量
	historyLen := len(kpiHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.KPI, historyLen-startIndex)
	copy(result, kpiHistorys[startIndex:])
	return result
}

// KPIHistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func KPIHistorySetSize(newSize int) {
	if newSize <= 0 {
		return // 无效的大小，不做任何修改
	}

	// 先更新最大记录数
	kpiHistorysMaxSizeMux.Lock()
	oldSize := kpiHistorysMaxSize
	kpiHistorysMaxSize = newSize
	kpiHistorysMaxSizeMux.Unlock()

	// 如果新的最大数量小于旧的最大数量，可能需要清理历史记录
	if newSize < oldSize {
		kpiHistorysMux.Lock()
		defer kpiHistorysMux.Unlock()
		// 如果历史记录数超过最大允许数量，只保留最新的记录
		if len(kpiHistorys) > kpiHistorysMaxSize {
			kpiHistorys = kpiHistorys[len(kpiHistorys)-kpiHistorysMaxSize:]
		}
	}
}

// safeAppendHistory 线程安全地添加历史记录
func safeAppendHistory(kpi model.KPI) {
	kpiHistorysMux.Lock()
	defer kpiHistorysMux.Unlock()

	// 获取最大历史记录数
	kpiHistorysMaxSizeMux.RLock()
	maxSize := kpiHistorysMaxSize
	kpiHistorysMaxSizeMux.RUnlock()

	if len(kpiHistorys) >= maxSize {
		// 如果超过，删除最旧的记录（索引为0的记录）
		kpiHistorys = kpiHistorys[1:]
	}

	kpiHistorys = append(kpiHistorys, kpi)
}

// KPISend 发送KPI
func KPISend(url, neUid string, granularity int64, dataMap map[string]float64) error {
	k := model.KPI{
		Data:        dataMap,
		Granularity: granularity,
		RecordTime:  time.Now().UnixMilli(),
		NeUid:       neUid,
	}

	safeAppendHistory(k)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := fetch.PostJSON(ctx, kpiUrl(url), k, nil)
	if err != nil {
		return err
	}
	return nil
}

func kpiUrl(url string) string { return url }
