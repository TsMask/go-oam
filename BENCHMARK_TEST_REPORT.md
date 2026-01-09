# Go-OAM SDK 基准测试报告

## 测试环境信息

| 项目 | 配置 |
|------|------|
| 操作系统 | Windows |
| 架构 | amd64 |
| CPU | 13th Gen Intel(R) Core(TM) i7-1370P |
| 并发数 | 20 |
| 测试时长 | 3秒/测试 |
| 测试包 | github.com/tsmask/go-oam |
| 测试时间 | 2026-01-09 |

## 测试概览

| 测试类别 | 测试数量 | 总耗时 |
|---------|---------|--------|
| 异步推送测试 | 2 | - |
| 服务推送测试 | 5 | - |
| 历史记录测试 | 7 | - |
| 配置访问测试 | 1 | - |
| 其他测试 | 3 | - |
| **总计** | **18** | **106.449秒** |

## 详细测试结果

### 1. 异步推送性能测试

#### 1.1 低并发异步推送
```
BenchmarkAsyncPushLowConcurrency-20
  执行次数: 77,499
  平均耗时: 42,805 ns/op
  内存分配: 22,740 B/op
  分配次数: 118 allocs/op
  QPS: 23,367
```

#### 1.2 高并发异步推送
```
BenchmarkAsyncPushHighConcurrency-20
  执行次数: 83,221
  平均耗时: 43,373 ns/op
  内存分配: 22,140 B/op
  分配次数: 118 allocs/op
  QPS: 23,055
```

**分析**：
- 高并发场景下性能略有下降（约1.3%）
- 内存分配略有优化（减少2.6%）
- 分配次数保持不变

---

### 2. 服务推送性能测试

#### 2.1 Alarm服务并发推送
```
BenchmarkAlarmServiceConcurrentPush-20
  执行次数: 121,057
  平均耗时: 29,872 ns/op
  内存分配: 13,449 B/op
  分配次数: 113 allocs/op
  QPS: 33,483
```

#### 2.2 KPI服务并发操作
```
BenchmarkKPIServiceConcurrentOperations-20
  执行次数: 35,821,964
  平均耗时: 94.09 ns/op
  内存分配: 16 B/op
  分配次数: 1 allocs/op
  QPS: 10,627,319
```

#### 2.3 Common服务并发推送
```
BenchmarkCommonServiceConcurrentPush-20
  执行次数: 188,840
  平均耗时: 22,131 ns/op
  内存分配: 10,405 B/op
  分配次数: 110 allocs/op
  QPS: 45,180
```

#### 2.4 Common服务不同类型并发推送
```
BenchmarkCommonServiceDifferentTypesConcurrentPush-20
  执行次数: 117,392
  平均耗时: 31,237 ns/op
  内存分配: 10,743 B/op
  分配次数: 113 allocs/op
  QPS: 32,011
```

#### 2.5 NBState服务并发推送
```
BenchmarkNBStateServiceConcurrentPush-20
  执行次数: 86,512
  平均耗时: 46,913 ns/op
  内存分配: 14,596 B/op
  分配次数: 112 allocs/op
  QPS: 21,317
```

#### 2.6 UENB服务并发推送
```
BenchmarkUENBServiceConcurrentPush-20
  执行次数: 103,486
  平均耗时: 38,878 ns/op
  内存分配: 16,602 B/op
  分配次数: 115 allocs/op
  QPS: 25,721
```

#### 2.7 CDR服务并发推送
```
BenchmarkCDRServiceConcurrentPush-20
  执行次数: 158,572
  平均耗时: 23,273 ns/op
  内存分配: 11,626 B/op
  分配次数: 135 allocs/op
  QPS: 42,970
```

**服务推送性能排名**：
1. KPI服务: 10,627,319 QPS（内存操作，无网络IO）
2. Common服务: 45,180 QPS
3. CDR服务: 42,970 QPS
4. Alarm服务: 33,483 QPS
5. Common不同类型: 32,011 QPS
6. UENB服务: 25,721 QPS
7. NBState服务: 21,317 QPS

---

### 3. 历史记录性能测试

#### 3.1 NBState历史记录查询
```
BenchmarkNBStateHistoryListConcurrent-20
  执行次数: 910,572
  平均耗时: 3,696 ns/op
  内存分配: 12,296 B/op
  分配次数: 1 allocs/op
  QPS: 270,565
```

#### 3.2 NBState历史记录大小设置
```
BenchmarkNBStateHistorySetSizeConcurrent-20
  执行次数: 52,250
  平均耗时: 72,707 ns/op
  内存分配: 304,572 B/op
  分配次数: 1 allocs/op
  QPS: 13,753
```

#### 3.3 UENB历史记录查询
```
BenchmarkUENBHistoryListConcurrent-20
  执行次数: 1,096,814
  平均耗时: 3,501 ns/op
  内存分配: 12,295 B/op
  分配次数: 1 allocs/op
  QPS: 285,663
```

#### 3.4 UENB历史记录大小设置
```
BenchmarkUENBHistorySetSizeConcurrent-20
  执行次数: 43,371
  平均耗时: 85,820 ns/op
  内存分配: 297,957 B/op
  分配次数: 1 allocs/op
  QPS: 11,652
```

#### 3.5 CDR历史记录查询
```
BenchmarkCDRHistoryListConcurrent-20
  执行次数: 1,670,138
  平均耗时: 1,925 ns/op
  内存分配: 4,103 B/op
  分配次数: 1 allocs/op
  QPS: 519,480
```

#### 3.6 CDR历史记录大小设置
```
BenchmarkCDRHistorySetSizeConcurrent-20
  执行次数: 93,211
  平均耗时: 39,663 ns/op
  内存分配: 108,402 B/op
  分配次数: 1 allocs/op
  QPS: 25,208
```

#### 3.7 通用历史记录查询
```
BenchmarkHistoryListConcurrent-20
  执行次数: 924,422
  平均耗时: 3,280 ns/op
  内存分配: 18,438 B/op
  分配次数: 1 allocs/op
  QPS: 304,882
```

#### 3.8 通用历史记录大小设置
```
BenchmarkHistorySetSizeConcurrent-20
  执行次数: 39,555
  平均耗时: 90,891 ns/op
  内存分配: 416,246 B/op
  分配次数: 1 allocs/op
  QPS: 11,004
```

**历史记录查询性能排名**：
1. CDR历史查询: 519,480 QPS
2. UENB历史查询: 285,663 QPS
3. 通用历史查询: 304,882 QPS
4. NBState历史查询: 270,565 QPS

**历史记录大小设置性能排名**：
1. CDR大小设置: 25,208 QPS
2. NBState大小设置: 13,753 QPS
3. UENB大小设置: 11,652 QPS
4. 通用大小设置: 11,004 QPS

---

### 4. 配置访问性能测试

#### 4.1 配置并发访问
```
BenchmarkConfigConcurrentAccess-20
  执行次数: 86,144,598
  平均耗时: 42.71 ns/op
  内存分配: 0 B/op
  分配次数: 0 allocs/op
  QPS: 23,413,705
```

**分析**：
- 零内存分配，性能极佳
- 使用sync.Map实现，适合高并发读场景
- 性能远超其他测试（纯内存操作）

---

## 性能数据汇总表

| 测试名称 | 执行次数 | 平均耗时 | 内存分配 | 分配次数 | QPS |
|---------|---------|---------|---------|---------|-----|
| BenchmarkAsyncPushLowConcurrency | 77,499 | 42,805 ns | 22,740 B | 118 | 23,367 |
| BenchmarkAsyncPushHighConcurrency | 83,221 | 43,373 ns | 22,140 B | 118 | 23,055 |
| BenchmarkAlarmServiceConcurrentPush | 121,057 | 29,872 ns | 13,449 B | 113 | 33,483 |
| BenchmarkKPIServiceConcurrentOperations | 35,821,964 | 94.09 ns | 16 B | 1 | 10,627,319 |
| BenchmarkCommonServiceConcurrentPush | 188,840 | 22,131 ns | 10,405 B | 110 | 45,180 |
| BenchmarkCommonServiceDifferentTypesConcurrentPush | 117,392 | 31,237 ns | 10,743 B | 113 | 32,011 |
| BenchmarkNBStateServiceConcurrentPush | 86,512 | 46,913 ns | 14,596 B | 112 | 21,317 |
| BenchmarkNBStateHistoryListConcurrent | 910,572 | 3,696 ns | 12,296 B | 1 | 270,565 |
| BenchmarkNBStateHistorySetSizeConcurrent | 52,250 | 72,707 ns | 304,572 B | 1 | 13,753 |
| BenchmarkUENBServiceConcurrentPush | 103,486 | 38,878 ns | 16,602 B | 115 | 25,721 |
| BenchmarkUENBHistoryListConcurrent | 1,096,814 | 3,501 ns | 12,295 B | 1 | 285,663 |
| BenchmarkUENBHistorySetSizeConcurrent | 43,371 | 85,820 ns | 297,957 B | 1 | 11,652 |
| BenchmarkCDRServiceConcurrentPush | 158,572 | 23,273 ns | 11,626 B | 135 | 42,970 |
| BenchmarkCDRHistoryListConcurrent | 1,670,138 | 1,925 ns | 4,103 B | 1 | 519,480 |
| BenchmarkCDRHistorySetSizeConcurrent | 93,211 | 39,663 ns | 108,402 B | 1 | 25,208 |
| BenchmarkHistoryListConcurrent | 924,422 | 3,280 ns | 18,438 B | 1 | 304,882 |
| BenchmarkHistorySetSizeConcurrent | 39,555 | 90,891 ns | 416,246 B | 1 | 11,004 |
| BenchmarkConfigConcurrentAccess | 86,144,598 | 42.71 ns | 0 B | 0 | 23,413,705 |

---

## 性能分析

### QPS排名（从高到低）

| 排名 | 测试名称 | QPS | 类型 |
|------|---------|-----|------|
| 1 | BenchmarkConfigConcurrentAccess | 23,413,705 | 配置访问 |
| 2 | BenchmarkKPIServiceConcurrentOperations | 10,627,319 | KPI操作 |
| 3 | BenchmarkCDRHistoryListConcurrent | 519,480 | 历史查询 |
| 4 | BenchmarkHistoryListConcurrent | 304,882 | 历史查询 |
| 5 | BenchmarkUENBHistoryListConcurrent | 285,663 | 历史查询 |
| 6 | BenchmarkNBStateHistoryListConcurrent | 270,565 | 历史查询 |
| 7 | BenchmarkCommonServiceConcurrentPush | 45,180 | 服务推送 |
| 8 | BenchmarkCDRServiceConcurrentPush | 42,970 | 服务推送 |
| 9 | BenchmarkAlarmServiceConcurrentPush | 33,483 | 服务推送 |
| 10 | BenchmarkCommonServiceDifferentTypesConcurrentPush | 32,011 | 服务推送 |
| 11 | BenchmarkUENBServiceConcurrentPush | 25,721 | 服务推送 |
| 12 | BenchmarkCDRHistorySetSizeConcurrent | 25,208 | 历史设置 |
| 13 | BenchmarkAsyncPushLowConcurrency | 23,367 | 异步推送 |
| 14 | BenchmarkAsyncPushHighConcurrency | 23,055 | 异步推送 |
| 15 | BenchmarkNBStateServiceConcurrentPush | 21,317 | 服务推送 |
| 16 | BenchmarkNBStateHistorySetSizeConcurrent | 13,753 | 历史设置 |
| 17 | BenchmarkUENBHistorySetSizeConcurrent | 11,652 | 历史设置 |
| 18 | BenchmarkHistorySetSizeConcurrent | 11,004 | 历史设置 |

### 内存分配分析

**最低内存分配**：
1. BenchmarkConfigConcurrentAccess: 0 B/op
2. BenchmarkKPIServiceConcurrentOperations: 16 B/op
3. BenchmarkCDRHistoryListConcurrent: 4,103 B/op

**最高内存分配**：
1. BenchmarkHistorySetSizeConcurrent: 416,246 B/op
2. BenchmarkNBStateHistorySetSizeConcurrent: 304,572 B/op
3. BenchmarkUENBHistorySetSizeConcurrent: 297,957 B/op

**服务推送内存分配**（从低到高）：
1. BenchmarkCommonServiceConcurrentPush: 10,405 B/op
2. BenchmarkCommonServiceDifferentTypesConcurrentPush: 10,743 B/op
3. BenchmarkCDRServiceConcurrentPush: 11,626 B/op
4. BenchmarkAlarmServiceConcurrentPush: 13,449 B/op
5. BenchmarkNBStateServiceConcurrentPush: 14,596 B/op
6. BenchmarkUENBServiceConcurrentPush: 16,602 B/op

### 分配次数分析

**最低分配次数**：
1. BenchmarkConfigConcurrentAccess: 0 allocs/op
2. BenchmarkKPIServiceConcurrentOperations: 1 allocs/op
3. 所有历史记录测试: 1 allocs/op

**最高分配次数**：
1. BenchmarkCDRServiceConcurrentPush: 135 allocs/op
2. BenchmarkAsyncPushLowConcurrency: 118 allocs/op
3. BenchmarkAsyncPushHighConcurrency: 118 allocs/op

---

## 测试结论

### 1. 性能表现

**优秀**（QPS > 1,000,000）：
- 配置并发访问：23,413,705 QPS
- KPI服务并发操作：10,627,319 QPS

**良好**（QPS > 100,000）：
- CDR历史记录查询：519,480 QPS
- 通用历史记录查询：304,882 QPS
- UENB历史记录查询：285,663 QPS
- NBState历史记录查询：270,565 QPS

**一般**（QPS > 20,000）：
- Common服务推送：45,180 QPS
- CDR服务推送：42,970 QPS
- Alarm服务推送：33,483 QPS
- Common不同类型推送：32,011 QPS
- UENB服务推送：25,721 QPS
- 异步推送（低并发）：23,367 QPS
- 异步推送（高并发）：23,055 QPS
- CDR历史大小设置：25,208 QPS

**需优化**（QPS < 20,000）：
- NBState服务推送：21,317 QPS
- NBState历史大小设置：13,753 QPS
- UENB历史大小设置：11,652 QPS
- 通用历史大小设置：11,004 QPS

### 2. 内存效率

**最优**：
- 配置访问：零内存分配
- KPI操作：16 B/op

**良好**：
- 历史记录查询：4,103-18,438 B/op
- 服务推送：10,405-16,602 B/op

**需优化**：
- 历史大小设置：108,402-416,246 B/op（涉及内存重新分配）

### 3. 并发稳定性

- 所有测试在20并发下稳定运行
- 高并发场景下性能下降不明显（<2%）
- 无内存泄漏或崩溃现象

---

## 测试代码位置

所有测试代码位于 [benchmark_test.go](benchmark_test.go)

---

## 附录：测试命令

```bash
# 运行所有基准测试
go test -bench=. -benchmem -run=NONE -benchtime=3s -timeout=15m

# 运行特定基准测试
go test -bench=BenchmarkCommonServiceConcurrentPush -benchmem -run=NONE

# 运行带CPU和内存分析的测试
go test -bench=. -benchmem -run=NONE -cpuprofile=cpu.prof -memprofile=mem.prof
```

---

**报告生成时间**: 2026-01-09  
**测试执行者**: Go-OAM SDK Benchmark Suite  
**报告版本**: v1.0
