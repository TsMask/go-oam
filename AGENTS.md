# AGENTS.md — go-oam 项目指南

## 项目概述

go-oam 是一个 Go 语言编写的 OAM（运维管理）SDK，用于网元（NE）与网管（NMS）之间的交互。模块路径 `github.com/tsmask/go-oam`，Go 1.25+。

## 构建与测试

```bash
go build ./...
go test ./...
go test ./ws/...
go test ./push/...

# Protobuf 生成（需要时）
protoc --go_out=. --go-grpc_out=. ws/protocol/ws.proto
```

## 代码风格

### 语言

- 注释、提交信息、文档使用**中文**
- 标识符（函数名、变量名、类型名）使用**英文**
- 日志信息使用中文

### 设计模式

- 统一使用 **Functional Options** 模式配置模块，`Option` 类型为 `func(*Config)`
- 选项函数命名：`WithXxx`，如 `WithTimeout`、`WithServerMaxConns`
- 内部配置结构体小写不导出（如 `serverConfig`、`clientConfig`），通过 Option 设置
- 构造函数统一 `New(opts ...Option)` 命名

### 并发安全

- 高频 map 使用分片锁减少竞争：`push/history` 和 `push/metrics` 使用 16 分片
- 计数器使用 `sync/atomic` 包的原子类型（如 `atomic.Int64`、`atomic.Int32`、`atomic.Bool`）
- 连接管理使用 `sync.RWMutex` 读写锁
- 关闭操作使用 `sync.Once` 防止重复

### 代码组织

- 每个文件使用 `// ===` 分隔线划分区域（类型定义、Option、方法实现等）
- 导出类型需要完整的 GoDoc 注释
- 中文注释风格：`// WithXxx 设置xxx` 简洁描述功能
- 非必要不添加抽象，保持"简单高性能"

### 用户偏好

- `go build` 然后运行二进制，不用 `go run`（慢）
- 浏览器测试时 Ctrl+Shift+R 强制刷新
- 用 `fuser -k <port>/tcp` 清理残留进程
- 服务端后台运行 `setsid ./wsServer &`
- 浏览器通过 IP 连接（`ws://192.168.9.58:9092/ws`），不用 localhost

## 目录结构

```
go-oam/
├── oam.go                      # SDK 入口
├── ws/                         # WebSocket 通信框架
│   ├── ws.go                   # 顶层 Facade（re-export + Option + 构造函数）
│   ├── ws_types.go             # Request/Response 类型 re-export
│   ├── server/
│   │   ├── server.go           # Server、ConnManager、topicManager
│   │   ├── conn.go             # Conn（readLoop/writeLoop/healthLoop，自动检测编码）
│   │   └── option.go           # ServerOption
│   ├── client/
│   │   ├── client.go           # Client（双层 context、自动重连、异步回调）
│   │   └── option.go           # ClientOption
│   ├── codec/
│   │   ├── codec.go            # Codec 接口、NewCodec 工厂、JSON 单例
│   │   ├── json.go             # JSON 编解码器
│   │   ├── msgpack.go          # MsgPack 编解码器
│   │   └── protobuf.go         # Protobuf 编解码器
│   ├── protocol/               # Protobuf 定义（ws.proto → ws.pb.go）
│   └── types/
│       └── message.go          # Request/Response 结构体（Data 为 json.RawMessage）
├── push/                       # 数据推送框架
│   ├── push.go                 # Push 核心客户端、Record、Option、工厂方法
│   ├── client/
│   │   └── client.go           # HTTP 客户端（Worker 池、异步队列、指数退避重试）
│   ├── history/
│   │   ├── history.go          # History 泛型历史记录（sync.Map + RingBuffer）
│   │   ├── ringbuffer.go       # RingBuffer 环形缓冲区
│   │   └── sharded.go          # ShardedHistory 分片历史记录（16 分片 + FNV-1a）
│   ├── metrics/
│   │   ├── metrics.go          # Metrics 指标采集（sync.Map）
│   │   └── sharded.go          # ShardedMetrics 分片指标采集（16 分片）
│   └── timer/
│       └── timer.go            # Timer 周期定时器
├── pkg/                        # 工具包（纯工具库，不依赖其他内部模块）
│   ├── cmd/                    # 本地命令行执行（exec、session、check）
│   ├── crypto/                 # AES 加密、哈希
│   ├── date/                   # 日期解析（字符串/数字转时间）
│   ├── fetch/                  # HTTP 请求封装（resty，含异步队列）
│   ├── file/                   # 文件操作（列表、上传、压缩、CSV/JSON/TXT，跨平台）
│   ├── generate/               # ID 生成（crypto/rand）
│   ├── iperf/                  # iperf 网络测试（PTY 终端封装）
│   ├── parse/                  # 数据解析
│   ├── ping/                   # ping 功能（原生 + PTY 终端两种实现）
│   ├── push/                   # 推送数据结构定义（alarm、cdr、kpi、nb_state、ue_ims、ue_nb）
│   ├── ringbuffer/             # 环形缓冲区
│   ├── socket/                 # TCP/UDP 客户端与服务端
│   ├── state/                  # 系统状态采集（CPU/内存/磁盘/网络/进程/监控）
│   └── telnet/                 # Telnet 客户端与服务端
└── examples/                   # 使用示例
    ├── ws/
    │   ├── web/                # WebSocket 服务端 + 浏览器测试页面
    │   │   ├── main.go         # 完整服务端示例（echo/broadcast/订阅/发布）
    │   │   └── index.html      # 浏览器客户端（消息收发 + 性能测试 + 发布订阅）
    │   └── client/             # WebSocket Go 客户端示例
    └── push/
        ├── usage/              # 功能测试（170 断言，覆盖全部 API）
        └── stats/              # 性能基准测试（10 项 benchmark + 正确性验证）
```

## 模块关系

- `pkg/` 是纯工具库，不依赖其他内部模块，可独立使用
- `ws/` 是独立的 WebSocket 通信框架，依赖 `pkg/generate`（ID 生成）
- `push/` 是独立的数据推送框架，依赖 `pkg/fetch`、`pkg/ringbuffer`
- `ws/` 和 `push/` 互不依赖
- `oam.go` 是 SDK 顶层入口

## 关键依赖

| 依赖 | 用途 |
|---|---|
| `coder/websocket` | WebSocket 协议实现 |
| `go-resty/resty/v2` | HTTP 客户端 |
| `vmihailenco/msgpack/v5` | MsgPack 编解码 |
| `google.golang.org/protobuf` | Protobuf 序列化 |
| `shirou/gopsutil/v4` | 系统状态采集 |
| `creack/pty` | PTY 终端（iperf/ping/telnet） |
| `prometheus-community/pro-bing` | Ping 功能 |
