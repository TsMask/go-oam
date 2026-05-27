# go-oam

[![Go Reference](https://pkg.go.dev/badge/github.com/olekukonko/tablewriter.svg)](https://pkg.go.dev/github.com/tsmask/go-oam)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsmask/go-oam)](https://goreportcard.com/report/github.com/tsmask/go-oam)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg)](LICENSE)
[![Tag](https://img.shields.io/badge/TAG-list-success)](https://proxy.golang.org/github.com/tsmask/go-oam/@v/list)

Go OAM SDK — 网元（NE）与网管（NMS）之间的运维通信框架。

提供 WebSocket、数据推送、系统状态采集、命令执行等完整能力，模块独立可组合。

## 模块概览

| 模块 | 说明 |
|---|---|
| `ws/` | WebSocket 通信框架（服务端 + 客户端，多编码自动检测，发布订阅） |
| `push/` | 数据推送框架（同步/异步，Worker 池，重试，指标采集，历史记录） |
| `pkg/` | 工具包（独立模块，不依赖其他内部包） |

## 快速开始

要求 Go 1.25+。

```bash
go get github.com/tsmask/go-oam
```

### WebSocket 服务端

```go
import "github.com/tsmask/go-oam/ws"

server := ws.NewServer(
    ws.WithServerCodec("json"),
    ws.WithServerHeartbeat(30*time.Second),
)

server.Handle("echo", func(conn *ws.Conn, req *ws.Request) {
    conn.SendOK(req.ID, req.Action, req.Data)
})

server.OnConnect(func(conn *ws.Conn, r *http.Request) {
    log.Printf("连接: %s", conn.ID())
})

mux := http.NewServeMux()
mux.Handle("/ws", server)
http.ListenAndServe(":9092", mux)
```

### WebSocket 客户端

```go
client := ws.NewClient("ws://localhost:9092/ws",
    ws.WithClientAutoReconnect(true),
)
client.Connect(context.Background())
defer client.Close()

client.OnReceive(func(resp *ws.Response) {
    fmt.Printf("收到: id=%s code=%d\n", resp.ID, resp.Code)
})

client.Send(&ws.Request{Action: "echo", Data: []byte(`"hello"`)})
```

### 数据推送

```go
import "github.com/tsmask/go-oam/push"

p := push.New(
    push.WithBaseURL("http://localhost:8080"),
    push.WithTimeout(30*time.Second),
    push.WithRetry(3),
)
defer p.Close()

record := &push.Record{
    NeUID:      "ne-001",
    RecordType: "alarm",
    RecordData: json.RawMessage(`{"level":"critical"}`),
}

p.Send(record, nil)      // 同步
p.SendAsync(record, nil)  // 异步
```

## 模块详情

### ws/ — WebSocket 通信框架

基于 [coder/websocket](https://github.com/coder/websocket)，支持服务端和客户端。

- **多编码自动检测** — 同一服务端同时服务 JSON/MsgPack/Protobuf 客户端，按消息类型自动切换
- **发布订阅** — 内置 Topic 管理，Subscribe/Publish/Broadcast
- **中间件** — 洋葱模型，按注册顺序包裹 Handler
- **元数据** — 每连接 SetMeta/GetMeta，线程安全
- **客户端自动重连** — 指数退避 + 抖动
- **心跳保活** — 连续 3 次 Ping 失败断开
- **优雅关闭** — Shutdown() 拒绝新连接 → 关闭所有已有连接

→ 完整文档见 [ws/README.md](ws/README.md)

### push/ — 数据推送框架

HTTP 推送客户端，支持同步/异步、重试、指标采集和历史记录。

- **Worker 池** — 可配置 Worker 数和队列大小，队列满时自动降级为同步发送
- **指数退避重试** — 可重试 5xx/429/网络错误，不可重试 4xx
- **Metrics** — 标准版和分片版两种指标采集实现
- **History** — 泛型环形缓冲区，标准和分片两种实现
- **Timer** — 周期回调定时器
- **连接池复用** — http.Client + sync.Pool 复用连接和 Buffer

→ 完整文档见 [push/README.md](push/README.md)

### pkg/ — 工具包

纯工具库，模块之间无依赖，可独立使用。

| 包 | 说明 |
|---|---|
| `pkg/cmd` | 本地命令行执行（exec、session、check） |
| `pkg/crypto` | AES 加密、哈希 |
| `pkg/date` | 日期解析（字符串/数字转时间） |
| `pkg/fetch` | HTTP 请求封装（基于 resty，含异步队列） |
| `pkg/file` | 文件操作（列表、上传、压缩、CSV/JSON/TXT，跨平台） |
| `pkg/generate` | ID 生成（crypto/rand） |
| `pkg/iperf` | iperf 网络测试（PTY 终端封装） |
| `pkg/parse` | 数据解析 |
| `pkg/ping` | ping 功能（原生 + PTY 终端两种实现） |
| `pkg/push` | 推送数据结构定义（alarm、cdr、kpi、nb_state、ue_ims、ue_nb） |
| `pkg/ringbuffer` | 环形缓冲区 |
| `pkg/socket` | TCP/UDP 客户端与服务端 |
| `pkg/state` | 系统状态采集（CPU/内存/磁盘/网络/进程） |
| `pkg/telnet` | Telnet 客户端与服务端 |

## 示例

```
examples/
├── push/
│   ├── usage/     功能测试（Record、Client、Metrics、History、Timer、综合场景）
│   └── stats/     性能基准测试（10 项 benchmark + 正确性验证）
└── ws/
    ├── web/       WebSocket 服务端 + 浏览器客户端
    └── client/    WebSocket Go 客户端
```

## 关键依赖

| 依赖 | 用途 |
|---|---|
| `coder/websocket` | WebSocket 协议实现 |
| `go-resty/resty/v2` | HTTP 客户端 |
| `vmihailenco/msgpack/v5` | MsgPack 编解码 |
| `google.golang.org/protobuf` | Protobuf 序列化 |
| `shirou/gopsutil/v4` | 系统状态采集 |
| `creack/pty` | PTY 终端 |
| `prometheus-community/pro-bing` | Ping 功能 |

## 构建

```bash
go build ./...

# 运行测试
go test ./...

# 运行示例
go run examples/ws/web/main.go
go run examples/push/usage/main.go
go run examples/push/stats/main.go 50  # 50 并发
```

## 目录结构

```
go-oam/
├── oam.go                  # SDK 入口
├── ws/                     # WebSocket 模块
│   ├── server/             # 服务端（连接管理、发布订阅、中间件）
│   ├── client/             # 客户端（自动重连、心跳）
│   ├── codec/              # 编解码器（JSON/MsgPack/Protobuf）
│   ├── types/              # Request/Response 消息结构
│   └── protocol/           # Protobuf 定义
├── push/                   # 推送模块
│   ├── client/             # HTTP 客户端（Worker 池、异步队列、重试）
│   ├── history/            # 历史记录（RingBuffer，标准/分片）
│   ├── metrics/            # 指标采集（标准/分片）
│   └── timer/              # 周期定时器
├── pkg/                    # 工具包（独立模块）
│   ├── cmd/                # 命令执行
│   ├── crypto/             # 加密
│   ├── date/               # 日期工具
│   ├── fetch/              # HTTP 请求
│   ├── file/               # 文件操作
│   ├── generate/           # ID 生成
│   ├── iperf/              # 网络测试
│   ├── parse/              # 数据解析
│   ├── ping/               # Ping
│   ├── push/               # 推送数据结构
│   ├── ringbuffer/         # 环形缓冲区
│   ├── socket/             # TCP/UDP
│   ├── state/              # 系统状态
│   └── telnet/             # Telnet
└── examples/               # 使用示例
    ├── ws/                 # WebSocket 示例
    └── push/               # Push 示例
```

## License

[BSD 3-Clause](LICENSE)
