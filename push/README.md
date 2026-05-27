# Push 推送模块

高性能数据推送框架，支持同步/异步推送、Worker 池、指数退避重试、指标采集和历史记录。

## 特性

- **同步/异步推送** — `Send` 阻塞等待，`SendAsync` 非阻塞入队
- **Worker 池** — 可配置 Worker 数量和队列大小，队列满时自动降级为同步发送
- **指数退避重试** — 可配置重试次数，抖动避免惊群
- **指标采集** — 标准 `Metrics` 和分片 `ShardedMetrics` 两种实现，支持 Register/Inc/Dec/Flush/Snapshot
- **历史记录** — 泛型 `RingBuffer`，支持标准和分片两种实现，自动淘汰旧数据
- **定时器** — `Timer` 周期回调，用于定时采集和推送
- **连接池复用** — `http.Client` + `sync.Pool` 复用连接和 Buffer，减少 GC 压力
- **批量推送** — `BatchPush` 并发发送多条记录

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

// 同步发送
if err := p.Send(record, nil); err != nil {
    log.Printf("发送失败: %v", err)
}

// 异步发送（非阻塞，入队后立即返回）
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

p.Send(record, params)          // 同步发送
p.SendAsync(record, params)     // 异步发送（非阻塞）

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

m.Register(name, init, step, min, max)  // 注册指标
m.Inc(name)                             // +step
m.Dec(name)                             // -step
m.IncBy(name, delta)                    // +delta
m.Set(name, value)                      // 设值
m.Get(name)                             // 取值
m.GetDelta(name)                        // 自上次 Flush 的增量

m.Flush()                               // 返回增量，更新 sent
m.FlushAndReset()                       // 返回当前值，重置为 init
m.Snapshot()                            // 只读快照
m.Clear()                               // 重置所有
m.Count()                               // 指标数量
m.Keys()                                // 指标名称列表
```

### ShardedMetrics 分片指标

与 `Metrics` 接口一致，内部使用 16 分片 + FNV-1a 哈希，减少锁竞争。适合高并发写入场景。

```go
sm := push.NewShardedMetrics()
sm.Register("requests", 0, 1, 0, 1e9)
sm.Inc("requests")
sm.Flush()
```

### History 历史记录

```go
// 标准版（按 key 隔离，lazy 创建）
h := push.NewHistory[push.Record](1000)    // 每个环形缓冲区容量 1000
h.Push("alarm", record)                    // 按 key 写入
h.List("alarm", 10)                        // 取最近 10 条（按插入顺序）
h.List("alarm", 0)                         // 取全部
h.Clear("alarm")
h.Keys()                                   // 所有 key
h.SetSize(2048)                            // 调整所有缓冲区大小
h.SetSizeByKey("alarm", 2048)              // 调整单个 key 的缓冲区

// 分片版（16 分片，高并发优化）
sh := push.NewShardedHistory[push.Record](1000)
sh.Push("alarm", record)
sh.List("alarm", 10)
sh.BatchPush(func(r Record) string { return r.RecordType }, records)  // 批量写入
sh.Count("alarm")                          // 单 key 元素数
sh.CountAll()                              // 所有元素总数
sh.ClearAll()                              // 清空所有
sh.SetSize(2048)                           // 调整所有分片大小
```

### Client HTTP 客户端

底层 HTTP 客户端，可独立使用：

```go
cli := push.NewClient(
    client.WithTimeout(30*time.Second),
    client.WithRetry(3),
    client.WithWorkers(8),
    client.WithQueueSize(4096),
)
defer cli.Close()

cli.Push(url, payload)                     // 同步
cli.PushTimeout(url, payload, timeout)     // 同步 + 自定义超时
cli.AsyncPush(url, payload)                // 异步
cli.AsyncPushTimeout(url, payload, timeout)// 异步 + 自定义超时
cli.BatchPush(url, payloads)               // 批量并发

cli.Stats()                                // PoolStats{ActiveWorkers, QueueLength, TotalProcessed, FailedCount}
cli.HealthCheck()                          // 健康检查
cli.SetWorkers(n)                          // 运行时调整 Worker 数量
```

### Timer 定时器

```go
timer := push.NewTimer()
timer.Start(interval, func(t time.Time) {
    // 周期回调
})
timer.Stop()                               // 优雅停止
timer.IsRunning()                          // 运行状态
```

## 配置选项

### Push 选项

| Option             | 默认值                | 说明               |
| ------------------ | --------------------- | ------------------ |
| `WithBaseURL(url)` | `"http://localhost"`  | 推送服务基础地址   |
| `WithPushURI(uri)` | `"/api/push/receive"` | 推送路径           |
| `WithTimeout(d)`   | `1m`                  | 请求超时           |
| `WithRetry(n)`     | `0`                   | 重试次数，0 不重试 |

### Client 选项

| Option                               | 默认值   | 说明                       |
| ------------------------------------ | -------- | -------------------------- |
| `WithBaseURL(url)`                   | `""`     | 基础地址（高层 Push 使用） |
| `WithTimeout(d)`                     | `1m`     | 请求超时                   |
| `WithRetry(n)`                       | `0`      | 重试次数                   |
| `WithWorkers(n)`                     | `NumCPU` | Worker 池大小              |
| `WithQueueSize(n)`                   | `4096`   | 异步队列容量               |
| `WithAsyncQueue(workers, queueSize)` | —        | 同时设置 Worker 和队列     |

## 架构设计

### 异步推送流程

```
SendAsync() → asyncCh(队列) → Worker 1 → HTTP POST → 重试?
                          → Worker 2 → HTTP POST → 重试?
                          → Worker N → HTTP POST → 重试?
                          ↓ 队列满时
                       同步降级发送
```

### 重试策略

- 指数退避：初始 100ms，最大 30s，每次加抖动
- 可重试条件：5xx、429、网络超时、连接拒绝/重置
- 不可重试：4xx（除 429）、context 取消

### 连接复用

- `http.Transport` 连接池：每主机最大 100 连接，空闲超时 90s
- `sync.Pool` 复用 `http.Client`、`bytes.Buffer`、`pushJob` 对象
- JSON 编码禁用 HTML 转义（`SetEscapeHTML(false)`）

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
| 批量写入 | 逐条 Push                 | `BatchPush` 按分片聚合           |
| 适用场景 | 一般场景                  | 高吞吐写入                       |

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
