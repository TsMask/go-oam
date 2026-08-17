# Push 推送模块

高性能数据推送框架，支持同步/异步推送、Worker 池、指数退避重试、指标采集、历史记录和定时回调。

```go
import "github.com/tsmask/go-oam/push"
```

## 特性

- **同步/异步推送** — `Send` 阻塞等待结果，`SendAsync` 非阻塞入队
- **Worker 池** — 可配置 Worker 数量和队列容量，队列满时自动降级为同步发送
- **指数退避重试** — 初始 100ms、上限 30s、附加随机抖动；仅作用于同步发送路径
- **指标采集** — 标准 `Metrics` 与 16 分片 `ShardedMetrics`，支持边界约束与增量导出
- **历史记录** — 泛型环形缓冲区，标准版按 key 隔离，分片版面向高吞吐写入
- **定时器** — `Timer` 周期回调，用于定时采集和推送
- **连接复用** — `http.Transport` 连接池 + `sync.Pool` 复用 Client / Buffer / Job 对象
- **批量推送** — `BatchPush` 并发提交多条记录

## 快速开始

### 基本推送

```go
p := push.New(
    push.WithBaseURL("http://localhost:8080"),
    push.WithTimeout(30*time.Second),
    push.WithRetry(3),
)
defer p.Close()

record := &push.Record{
    NeUID:      "ne-001",
    RecordType: "alarm",
    RecordData: json.RawMessage(`{"level":"critical","message":"CPU overload"}`),
}

// 同步发送：失败时按 WithRetry 配置重试
if err := p.Send(record, nil); err != nil {
    log.Printf("发送失败: %v", err)
}

// 异步发送：入队成功即返回，Worker 投递失败不重试
if err := p.SendAsync(record, nil); err != nil {
    log.Printf("入队失败: %v", err)
}
```

### 定时采集推送

```go
p := push.New(push.WithBaseURL("http://localhost:8080"))
defer p.Close()

m := push.NewMetrics()
m.Register("cpu_usage", 0, 1, 0, 100)

timer := push.NewTimer()
timer.Start(5*time.Second, func(t time.Time) {
    m.IncBy("cpu_usage", 1.5)
    data := m.FlushAndReset()
    recordData, _ := json.Marshal(data)
    p.Send(&push.Record{
        NeUID:      "ne-001",
        RecordType: "metrics",
        RecordData: recordData,
    }, nil)
})

// 需要停止时
timer.Stop()
```

### 自定义目标地址

每次发送可通过 `SendParams` 指定不同的 URL 和超时：

```go
err := p.Send(record, &push.SendParams{
    URL:     "https://custom-api.com/hook",
    Timeout: 5 * time.Second,
})
```

## 发送语义

`Send` / `SendAsync` 的关键差异（按当前实现）：

| 行为 | Send（同步） | SendAsync（异步） |
|---|---|---|
| 返回时机 | 请求完成或超时后 | 编码并入队后立即返回 |
| 重试 | 使用 `WithRetry` 配置 | 不重试 |
| Timeout 含义 | 覆盖所有尝试及退避等待的总预算 | 单次投递的总预算 |
| 失败感知 | 返回 error | 入队成功后无法感知投递失败 |

补充说明：

- `Timeout` 是整次操作的总预算：重试期间的退避等待计入同一 context，超时后停止后续尝试。
- `Send` / `SendAsync` 会在 `RecordTime == 0` 时原地填充当前 UTC 毫秒时间戳，即修改传入的 `record`。
- `Close` 关闭队列并等待已入队任务全部执行完；`Close` 之后不要再调用 `SendAsync`，当前实现会因向已关闭通道发送而 panic。
- 队列满时的同步降级发送同样不重试。
- `BatchPush` 的"成功"指入队成功（或降级同步发送成功），返回时不保证对端已收到全部消息。

## 重试策略

- 指数退避：初始 100ms，每次翻倍，上限 30s；每次附加 0~50% 基准延迟的随机抖动。
- 可重试：5xx、429、网络超时、连接拒绝/重置、temporary failure。
- 不可重试：其余 4xx、context 取消/超时。
- 重试仅作用于同步路径：`Send`、`Push`、`PushTimeout`。异步队列中的任务与队列满降级发送均不重试。

## 消息格式

### Record 推送记录

```json
{
  "core_uid": "core-001",
  "ne_uid": "ne-001",
  "record_time": 1716700000000,
  "record_type": "alarm",
  "record_data": { "level": "critical" },
  "params": { "source": "agent" }
}
```

| 字段          | 类型            | 说明                                         |
| ------------- | --------------- | -------------------------------------------- |
| `core_uid`    | string          | Core 网络标识（可选）                        |
| `ne_uid`      | string          | 网元标识（可选）                             |
| `record_time` | int64           | 记录时间，UTC 毫秒。为 0 时自动填充当前时间  |
| `record_type` | string          | 记录类型，如 `"alarm"`、`"metrics"`、`"kpi"` |
| `record_data` | json.RawMessage | 业务数据，延迟解码                           |
| `params`      | map             | 附加参数（可选）                             |

## API 参考

### Push 核心客户端

```go
p := push.New(opts...)          // 创建推送客户端

p.Send(record, params)          // 同步发送（支持重试）
p.SendAsync(record, params)     // 异步发送（不重试）

p.Close()                       // 关闭，等待队列中任务完成
```

### SendParams 发送参数

```go
&push.SendParams{
    URL:     "",                 // 空 → 使用默认 baseURL + pushURI
    Timeout: 0,                  // ≤0 → 使用默认超时
}
```

### Metrics 指标采集

```go
m := push.NewMetrics()

m.Register(name, init, step, min, max)  // 注册；同名重复注册会覆盖并重置
m.Inc(name)                             // +step，超过 max 截断
m.Dec(name)                             // -step，低于 min 截断
m.IncBy(name, delta)                    // +delta，仅检查上界 max
m.Set(name, value)                      // 直接设值，不做边界约束
m.Get(name)                             // 取当前值，未注册返回 0
m.GetDelta(name)                        // 自上次 Flush 的增量

m.Flush()                               // 返回增量，更新 sent
m.FlushAndReset()                       // 返回当前值，重置为 init
m.Snapshot()                            // 只读快照
m.Clear()                               // 重置所有
m.Count()                               // 指标数量
m.Keys()                                // 指标名称列表
```

注意：

- 未注册的指标上调用 `Inc` / `Dec` / `IncBy` / `Set` 为空操作，`Get` / `GetDelta` 返回 0。
- `Register` 会覆盖同名指标并将其重置为新配置的初始值。

### ShardedMetrics 分片指标

API 与 `Metrics` 完全相同，内部使用 16 分片 + FNV-1a 哈希，减少锁竞争。上述注册覆盖、边界截断等语义同样适用。适合高并发写入场景。

```go
sm := push.NewShardedMetrics()
sm.Register("requests", 0, 1, 0, 1e9)
sm.Inc("requests")
sm.Flush()
```

### History

```go
// 标准版：按 key 严格隔离，lazy 创建缓冲区
h := push.NewHistory[push.Record](1000)    // 每个 key 的缓冲区容量 1000；≤0 时按 1024 处理
h.Push("alarm", record)                    // 按 key 写入
h.List("alarm", 10)                        // 最近 10 条（旧 → 新）
h.List("alarm", 0)                         // 全部
h.Clear("alarm")
h.Keys()                                   // 所有 key
h.SetSize(2048)                            // 调整所有缓冲区大小
h.SetSizeByKey("alarm", 2048)              // 调整单个 key 的缓冲区
```

容量满时自动覆盖最旧元素；缩小容量时保留最新数据。

### ShardedHistory 分片历史记录

```go
sh := push.NewShardedHistory[push.Record](1000)
sh.Push("alarm", record)
sh.List("alarm", 10)
sh.BatchPush(func(r push.Record) string { return r.RecordType }, records)
sh.Count("alarm")                          // 该 key 所在分片的元素数
sh.CountAll()                              // 所有元素总数
sh.ClearAll()
sh.SetSize(2048)
```

注意语义差异：`ShardedHistory` 将 key 经 FNV-1a 哈希到 16 个分片，每个分片只有一个环形缓冲区。哈希到同一分片的不同 key 共享缓冲区，`List` / `Count` 返回的是该分片的混合数据，不保证只包含指定 key。需要严格按 key 隔离时使用标准 `History`。

### Client HTTP 客户端

底层 HTTP 客户端，可独立使用（选项需导入 `github.com/tsmask/go-oam/push/client`）：

```go
import "github.com/tsmask/go-oam/push/client"

cli := client.New(
    client.WithTimeout(30*time.Second),
    client.WithRetry(3),
    client.WithWorkers(8),
    client.WithQueueSize(4096),
)
defer cli.Close()

cli.Push(url, payload)                     // 同步（按 WithRetry 重试）
cli.PushTimeout(url, payload, timeout)     // 同步 + 自定义总超时
cli.AsyncPush(url, payload)                // 异步（不重试）
cli.AsyncPushTimeout(url, payload, timeout)// 异步 + 自定义超时
cli.BatchPush(url, payloads)               // 并发提交，等待全部入队/降级发送返回

cli.Stats()                                // PoolStats
cli.HealthCheck()                          // 未运行时返回错误
cli.SetWorkers(n)                          // 运行时调整 Worker 数量
```

`PoolStats` 字段：

| 字段 | 说明 |
|---|---|
| `ActiveWorkers` | 当前存活的 Worker goroutine 数 |
| `QueueLength` | 异步队列中等待的任务数 |
| `TotalProcessed` | 异步任务投递成功累计 |
| `FailedCount` | 异步任务投递失败累计 |

统计仅覆盖异步队列任务；`Push` / `PushTimeout` 等同步发送不计入。

### Timer 定时器

```go
timer := push.NewTimer()
timer.Start(interval, func(t time.Time) {
    // 周期回调
})
timer.Stop()                               // 发送停止信号
timer.IsRunning()                          // 运行状态
```

- 运行中重复 `Start` 为空操作。
- `interval` 必须大于 0，否则底层 `time.NewTicker` 会 panic。
- `Stop` 只发送停止信号，不等待正在执行的回调返回；需要确保收尾完成时请在回调内自行同步。
- 当前实现的 `stopOnce` 只在首次 `Stop` 时执行清理：`Stop` 后再次 `Start` 的定时器将无法再次停止，回调会持续触发。需要周期性启停时，请为每个周期创建新的 `Timer` 实例。

## 配置选项

### Push 选项

| Option             | 默认值                | 说明                                                 |
| ------------------ | --------------------- | ---------------------------------------------------- |
| `WithBaseURL(url)` | `"http://localhost"`  | 推送服务基础地址，空值忽略                           |
| `WithPushURI(uri)` | `"/api/push/receive"` | 推送路径，空值忽略                                   |
| `WithTimeout(d)`   | `1m`                  | 单次发送操作总超时（含重试与退避等待），≤0 忽略      |
| `WithRetry(n)`     | `0`                   | 同步发送重试次数，0 不重试；异步发送不重试           |

### Client 选项

| Option                               | 默认值   | 说明                                    |
| ------------------------------------ | -------- | --------------------------------------- |
| `WithBaseURL(url)`                   | `""`     | 仅供高层封装使用，Client 方法不拼接     |
| `WithTimeout(d)`                     | `1m`     | 默认总超时                              |
| `WithRetry(n)`                       | `0`      | 同步发送重试次数                        |
| `WithWorkers(n)`                     | `NumCPU` | Worker 池大小                           |
| `WithQueueSize(n)`                   | `4096`   | 异步队列容量                            |
| `WithAsyncQueue(workers, queueSize)` | —        | 同时设置 Worker 数量和队列容量          |

## 架构设计

### 异步推送流程

```
SendAsync() → asyncCh(队列) → Worker 1 → HTTP POST（不重试）
                          → Worker 2 → HTTP POST（不重试）
                          → Worker N → HTTP POST（不重试）
                          ↓ 队列满时
                       同步降级发送（不重试）
```

### 连接复用

- `http.Transport`：`MaxIdleConns=100`、`MaxIdleConnsPerHost=10`、`MaxConnsPerHost=100`、`IdleConnTimeout=90s`
- `sync.Pool` 复用 `http.Client`、`bytes.Buffer`、`pushJob` 对象
- JSON 编码禁用 HTML 转义（`SetEscapeHTML(false)`）
- 请求为 HTTP POST，`Content-Type: application/json`，`User-Agent: go-oam-push/1.0`
- 错误响应体最多读取 4096 字节用于错误信息

### Metrics 选型

|          | Metrics               | ShardedMetrics            |
| -------- | --------------------- | ------------------------- |
| 底层存储 | `sync.Map`            | 16 分片 `map` + `RWMutex` |
| 并发写入 | 低并发 OK             | 高并发优化                |
| 适用场景 | < 100 指标，< 10 协程 | > 100 并发，实时监控      |

### History 选型

|          | History                   | ShardedHistory                   |
| -------- | ------------------------- | -------------------------------- |
| 底层存储 | `sync.Map` → `RingBuffer` | 16 分片 `RingBuffer` + `RWMutex` |
| key 隔离 | 严格按 key                | 同分片 key 混合存储              |
| 批量写入 | 逐条 Push                 | `BatchPush` 按分片聚合           |
| 适用场景 | 需要 key 级隔离           | 高吞吐、允许分片内混合           |

## 目录结构

```
push/
├── push.go                 # Push 核心客户端、Record、Option、工厂方法
├── client/
│   └── client.go           # HTTP 客户端（Worker 池、异步队列、重试）
├── history/
│   ├── history.go          # History 泛型历史记录（sync.Map + RingBuffer）
│   ├── ringbuffer.go       # RingBuffer 环形缓冲区
│   └── sharded.go          # ShardedHistory 分片历史记录（16 分片）
├── metrics/
│   ├── metrics.go          # Metrics 指标采集（sync.Map）
│   └── sharded.go          # ShardedMetrics 分片指标采集（16 分片）
└── timer/
    └── timer.go            # Timer 周期定时器
```

## 示例

完整可运行示例见仓库 `examples/push/` 目录：

- `examples/push/usage` — 全功能验证：Record 推送、Client、Metrics、History、Timer、重试、队列降级
- `examples/push/stats` — 性能基准：并发指标、历史、HTTP 客户端、Timer 与综合场景
