# go-oam

[![Go Reference](https://pkg.go.dev/badge/github.com/olekukonko/tablewriter.svg)](https://pkg.go.dev/github.com/tsmask/go-oam)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsmask/go-oam)](https://goreportcard.com/report/github.com/tsmask/go-oam)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg)](LICENSE)
[![Tag](https://img.shields.io/badge/TAG-list-success)](https://proxy.golang.org/github.com/tsmask/go-oam/@v/list)

go-oam 是面向网络运维场景的 Go SDK，提供 WebSocket 通信、HTTP 数据推送、系统状态采集、远程连接和通用文件工具。模块可以独立引入，也可以组合成 OAM Agent / NMS 之间的通信层。

## 环境要求

- Go 1.25+
- 主要目标平台：Linux、Windows、macOS
- 当前验证：Windows 与 Linux 测试通过，macOS 交叉编译通过

## 安装

```bash
go get github.com/tsmask/go-oam
```

## 模块概览

| 模块 | 说明 | 详细文档 |
|---|---|---|
| `ws/` | WebSocket 服务端与客户端，支持 JSON / MsgPack / Protobuf、发布订阅、中间件、心跳和自动重连 | [ws/README.md](ws/README.md) |
| `push/` | HTTP 推送框架，支持同步重试、异步队列、指标、历史记录和定时器 | [push/README.md](push/README.md) |
| `pkg/` | 可独立使用的基础工具包，不依赖 `ws/` 和 `push/` | [pkg/README.md](pkg/README.md) |
| root | `oam.New()` 入口，目前仅提供版本配置与查询 | [oam.go](oam.go) |

## 示例

```text
examples/
├── push/
│   ├── stats/     # 指标、历史、HTTP 客户端和 Timer 基准场景
│   └── usage/     # Push、Client、Metrics、History、Timer 综合示例
└── ws/
    ├── client/    # Go WebSocket 客户端
    ├── server/    # Go WebSocket 服务端
    └── web/       # 浏览器客户端与服务端示例
```

运行示例：

```bash
go run ./examples/ws/server
go run ./examples/ws/client
go run ./examples/ws/web
go run ./examples/push/usage
go run ./examples/push/stats
```

## 构建与测试

```bash
go build ./...
go test ./...
go vet ./...

# 并发测试
CGO_ENABLED=1 go test -race ./pkg/file

# 交叉编译
GOOS=linux GOARCH=amd64 go build ./...
GOOS=darwin GOARCH=arm64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
```

## 主要依赖

| 依赖 | 用途 |
|---|---|
| `github.com/coder/websocket` | WebSocket 协议 |
| `github.com/go-resty/resty/v2` | HTTP 客户端 |
| `github.com/vmihailenco/msgpack/v5` | MsgPack 编解码 |
| `google.golang.org/protobuf` | Protobuf 编解码 |
| `github.com/shirou/gopsutil/v4` | 系统状态采集 |
| `github.com/creack/pty` | Unix PTY |
| `github.com/pkg/sftp` 与 `golang.org/x/crypto/ssh` | SSH 与 SFTP |

## 目录结构

```text
go-oam/
├── oam.go             # SDK 入口与版本配置
├── ws/                 # WebSocket 模块
│   ├── server/         # 服务端、连接管理、发布订阅
│   ├── client/         # 客户端、状态机、自动重连
│   ├── codec/          # JSON / MsgPack / Protobuf 编解码
│   ├── protocol/       # Protobuf 定义与生成代码
│   └── types/          # Request / Response 类型
├── push/               # 数据推送模块
│   ├── client/         # HTTP 客户端、Worker 队列、重试
│   ├── history/        # 标准与分片历史记录
│   ├── metrics/        # 标准与分片指标
│   └── timer/          # 周期定时器
├── pkg/                # 独立工具包
│   ├── cmd/            # 命令执行与 PTY
│   ├── crypto/         # 加密与摘要
│   ├── date/           # 时间解析
│   ├── fetch/          # HTTP 请求封装
│   ├── file/           # 文件、上传、归档
│   ├── generate/       # 随机生成
│   ├── parse/          # 数据解析
│   ├── ringbuffer/     # 环形缓冲区
│   ├── socket/         # TCP / UDP
│   ├── ssh/            # SSH / SFTP / 交互终端
│   ├── state/          # 系统状态
│   └── telnet/         # Telnet
└── examples/           # ws 与 push 示例
```

## License

[BSD 3-Clause](LICENSE)
