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

// KPI 指标服务
type KPI struct {
	NeUid              string             // 网元唯一标识
	Granularity        time.Duration      // 指标缓存时间粒度
	data               sync.Map           // 存储string -> *atomic.Uint64
	clearMutex         sync.Mutex         // 保护清空操作
	kpiTimerCancel     context.CancelFunc // KPI 定时发送取消函数
	kpiHistorys        []model.KPI        // KPI历史记录
	kpiHistorysMux     sync.RWMutex       // 保护kpiHistorys的并发访问
	kpiHistorysMaxSize atomic.Int32       // 最大历史记录数量
}

// NewKPI 创建KPI服务
func NewKPI(neUid string, granularity time.Duration) *KPI {
	k := &KPI{
		NeUid:       neUid,
		Granularity: granularity,
		kpiHistorys: []model.KPI{},
	}
	k.kpiHistorysMaxSize.Store(4096)
	return k
}

// KPITimerStart KPI定时发送
func (s *KPI) KPITimerStart(url string) {
	if s == nil {
		return
	}
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
					err := s.Send(url, s.NeUid, granularity, dataMap)
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
	if s == nil {
		return
	}
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
func (s *KPI) KeyAdd(key string, v float64) {
	if s == nil {
		return
	}

	atomicVal := s.getOrCreateAtomicValue(key)
	vUint := uint64(math.Abs(v * precisionMultiplier))

	if v >= 0 {
		atomicVal.Add(vUint)
	} else {
		atomicVal.Add(^uint64(vUint - 1))
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

// HistoryList 线程安全地获取历史列表
// n 为返回的最大记录数，n<0返回空列表 n=0返回所有记录
func (s *KPI) HistoryList(n int) []model.KPI {
	if s == nil {
		return []model.KPI{}
	}
	s.kpiHistorysMux.RLock()
	defer s.kpiHistorysMux.RUnlock()

	if n < 0 {
		return []model.KPI{}
	}

	// 计算要返回的记录数量
	historyLen := len(s.kpiHistorys)
	startIndex := 0

	// 仅当 n > 0 并且历史记录数大于 n 时才截取
	if n > 0 && historyLen > n {
		startIndex = historyLen - n
	}

	// 只复制需要的部分
	result := make([]model.KPI, historyLen-startIndex)
	copy(result, s.kpiHistorys[startIndex:])
	return result
}

// HistorySetSize 安全地修改最大历史记录数量
// 如果新的最大数量小于当前记录数，会自动清理旧记录
func (s *KPI) HistorySetSize(newSize int) {
	if s == nil || newSize <= 0 {
		return
	}

	oldSize := s.kpiHistorysMaxSize.Swap(int32(newSize))
	if newSize < int(oldSize) {
		s.kpiHistorysMux.Lock()
		defer s.kpiHistorysMux.Unlock()
		if len(s.kpiHistorys) > newSize {
			s.kpiHistorys = s.kpiHistorys[len(s.kpiHistorys)-newSize:]
		}
	}
}

// safeAppendHistory 线程安全地添加历史记录
func (s *KPI) safeAppendHistory(kpi model.KPI) {
	if s == nil {
		return
	}
	s.kpiHistorysMux.Lock()
	defer s.kpiHistorysMux.Unlock()

	maxSize := s.kpiHistorysMaxSize.Load()
	if len(s.kpiHistorys) >= int(maxSize) {
		s.kpiHistorys = s.kpiHistorys[1:]
	}

	s.kpiHistorys = append(s.kpiHistorys, kpi)
}

// PushURL 推送KPI到指定URL
func (s *KPI) PushURL(url string) error {
	if s == nil {
		return nil
	}
	dataMap := s.safeGetAllData()
	if len(dataMap) == 0 {
		return nil
	}
	granularity := int64(s.Granularity.Seconds())
	err := s.Send(url, s.NeUid, granularity, dataMap)
	if err == nil {
		s.safeClearData()
	}
	return err
}

// Send 发送KPI
func (s *KPI) Send(url, neUid string, granularity int64, dataMap map[string]float64) error {
	k := model.KPI{
		Data:        dataMap,
		Granularity: granularity,
		RecordTime:  time.Now().UnixMilli(),
		NeUid:       neUid,
	}

	s.safeAppendHistory(k)

	// 发送推送请求
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	return fetch.AsyncPush(ctx, url, k)
}
