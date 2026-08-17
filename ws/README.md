# WebSocket 通信框架

高性能 WebSocket 通信框架，支持请求-响应模式、发布订阅、中间件和自动重连。基于 [coder/websocket](https://github.com/coder/websocket) 实现。

```go
import ws "github.com/tsmask/go-oam/ws"
```

## 特性

- **请求-响应** — 客户端异步发送，通过 `OnReceive` 回调按 `resp.ID` 匹配响应，不阻塞等待
- **多编码** — 内置 JSON / MsgPack / Protobuf 编解码器，服务端按 WebSocket 帧类型自动检测编码
- **发布订阅** — 内置 Topic 管理，`Subscribe` / `Unsubscribe` / `Publish` / `Broadcast` 及条件过滤一应俱全
- **中间件** — 洋葱模型，按注册顺序包裹 Handler
- **自动重连** — 指数退避 + 随机抖动，客户端内置，可配置最大重连次数
- **心跳保活** — 服务端/客户端均可配置，连续 3 次 Ping 失败后断开
- **优雅关闭** — `Shutdown()` 拒绝新连接（HTTP 503）并关闭所有已有连接
- **元数据** — 每连接 `SetMeta` / `GetMeta`，线程安全；连接建立时自动写入 `remote_addr`、`user_agent`、`connected_at`
- **非阻塞发送（服务端）** — 每连接发送缓冲区满时返回 `ErrSendFull`

## 快速开始

### 服务端

```go
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

func main() {
	server := ws.NewServer(
		ws.WithServerCodec("json"),
		ws.WithServerMaxConns(10000),
		ws.WithServerSendBufferSize(1000),
		ws.WithServerHeartbeat(30*time.Second),
		ws.WithServerMaxMessageSize(1<<20), // 0 表示不限制
	)

	// 中间件在 Handle 注册时包裹 Handler，建议先 Use 再 Handle
	server.Use(func(next ws.Handler) ws.Handler {
		return func(conn *ws.Conn, req *ws.Request) {
			start := time.Now()
			next(conn, req)
			log.Printf("[MW] %s %s %v", conn.ID()[:8], req.Action, time.Since(start))
		}
	})

	server.Handle("echo", func(conn *ws.Conn, req *ws.Request) {
		conn.SendResp(&ws.Response{
			ID:     req.ID,
			Action: req.Action,
			Code:   200,
			Data:   req.Data,
		})
	})

	// r 是原始 HTTP 请求，可读取 Header / Cookie 等
	server.OnConnect(func(conn *ws.Conn, r *http.Request) {
		log.Printf("连接: %s %s", conn.ID(), r.RemoteAddr)
	})
	server.OnDisconnect(func(conn *ws.Conn) {
		log.Printf("断开: %s", conn.ID())
	})

	mux := http.NewServeMux()
	mux.Handle("/ws", server)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		server.Shutdown()
	}()

	log.Fatal(http.ListenAndServe(":9092", mux))
}
```

### 客户端

```go
client := ws.NewClient("ws://localhost:9092/ws",
	ws.WithClientAutoReconnect(true),
	ws.WithClientMaxReconnectAttempts(10),
)

// 建议在 Connect 之前设置回调，避免错过早到的响应
client.OnReceive(func(resp *ws.Response) {
	log.Printf("收到响应: id=%s code=%d", resp.ID, resp.Code)
})
client.OnState(func(s ws.State) { log.Printf("状态: %s", s) })
client.OnError(func(err error) { log.Printf("错误: %v", err) })

if err := client.Connect(context.Background()); err != nil {
	log.Fatal(err)
}
defer client.Close()

// ID 留空时由客户端自动生成；发送不等响应，响应走 OnReceive
err := client.Send(&ws.Request{
	Action: "echo",
	Data:   []byte(`"hello"`),
})
```

## 编解码规则

| 方向 | 帧类型 | 编解码器 |
|---|---|---|
| 客户端 → 服务端 | 文本帧 | 始终 JSON |
| 客户端 → 服务端 | 二进制帧 | 服务端配置的编码器 |
| 服务端 → 客户端 | — | 跟随最近一次请求检测出的编码器 |

- JSON 走 WebSocket 文本帧；MsgPack / Protobuf 走二进制帧。
- 文本帧和二进制帧可以通过帧类型区分，因此同一服务端可同时服务 JSON 客户端和一种二进制编码客户端（`msgpack` 或 `protobuf`）。
- MsgPack 和 Protobuf 均为二进制帧，无法通过帧类型互相区分，二进制客户端需与服务端 `WithServerCodec` 配置一致。
- 客户端始终按自身配置发送；服务端会自动跟随客户端编码回复。

## 发布订阅

```go
// 服务端 — 在 Handler 中让连接订阅
server.Handle("subscribe", func(conn *ws.Conn, req *ws.Request) {
	conn.Subscribe("news", "alerts")
	conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200})
})

resp := &ws.Response{Action: "news", Code: 200, Data: []byte(`{"title":"hello"}`)}

// 向 topic 发布消息
server.Publish("news", resp)

// 条件发布
server.PublishFilter("news", resp, func(c *ws.Conn) bool {
	return c.ID() != senderID
})

// 广播所有连接
server.Broadcast(resp)

// 条件广播
server.BroadcastFilter(resp, func(c *ws.Conn) bool {
	return c.ID() != senderID
})

// 查询
server.Topics()           // []string — 所有有订阅者的 topic
server.TopicCount("news") // int — 订阅者数量
conn.Subscriptions()      // []string — 当前连接订阅的 topic
```

连接关闭时会自动清理其全部订阅。

## 并发模型

- 服务端每条消息在独立 goroutine 中执行 Handler，同一连接的多条消息也可能并发执行；Handler 访问共享状态需自行加锁。
- `Conn.SetMeta` / `GetMeta`、`Subscribe` / `Unsubscribe`、`Server.Handle` 内部已做线程安全处理。
- 中间件在 `Handle` 注册时包裹 Handler，之后新增的中间件不会影响已注册的 Handler，建议先 `Use` 再 `Handle`。

## 内置错误响应

| 场景 | Action | Code | 行为 |
|---|---|---|---|
| 消息解码失败 | `invalid_request` | 400 | 返回错误后继续读取后续消息 |
| 消息超过大小限制 | `invalid_request` | 413 | 返回错误后继续读取后续消息 |
| 未注册的 action | 原请求 action | 404 | 继续读取后续消息 |
| Handler panic | 原请求 action | 500 | recover 后返回，连接保持 |

## 与 Gin 集成

```go
r := gin.Default()

r.GET("/ws", func(c *gin.Context) {
	server.ServeHTTP(c.Writer, c.Request)
	// ServeHTTP 内部已完成协议升级，不能再用 gin.Context 写普通 HTTP 响应
})

// 鉴权等 HTTP 层信息在 OnConnect 中读取原始 *http.Request
server.OnConnect(func(conn *ws.Conn, r *http.Request) {
	conn.SetMeta("user", r.Header.Get("X-User"))
})
```

## API 参考

### Server

```go
server := ws.NewServer(opts...)          // 创建，默认 JSON

server.Use(middleware...)                // 注册中间件（影响之后 Handle 的处理器）
server.Handle(action, handler)           // 注册处理器（线程安全）

server.OnConnect(fn)                     // 连接回调 fn(*Conn, *http.Request)
server.OnDisconnect(fn)                  // 断开回调 fn(*Conn)

server.Broadcast(resp)                   // 广播所有连接（*Response）
server.BroadcastFilter(resp, fn)         // 条件广播
server.Publish(topic, resp)              // 向 topic 发布
server.PublishFilter(topic, resp, fn)    // 条件发布
server.Topics()                          // 有订阅者的 topic 列表
server.TopicCount(topic)                 // topic 订阅者数

server.ConnManager()                     // 连接管理器
server.Codec()                           // 编解码器
server.Shutdown()                        // 优雅关闭
server.ServeHTTP(w, r)                   // 实现 http.Handler
```

### Conn

```go
conn.ID()                    // 连接唯一 ID
conn.Context()               // context.Context，取消时连接关闭
conn.LastActiveTime()        // 最后活跃时间（读到消息或 Ping 成功时刷新）
conn.CodecName()             // 当前响应编码器名称

conn.SendResp(resp)          // 发送响应；Ts 自动填充为当前毫秒时间戳

conn.SetMeta(key, val)       // 设置元数据（val 为 nil 时删除）
conn.GetMeta(key)            // 获取元数据

conn.Subscribe(topics...)    // 订阅（幂等，重复订阅不报错）
conn.Unsubscribe(topics...)  // 取消订阅
conn.Subscriptions()         // 已订阅 topic 列表

conn.Close()                 // 关闭连接（幂等，发送 Close 帧并触发 OnDisconnect）
```

连接建立时服务端自动写入元数据：`remote_addr`、`user_agent`、`connected_at`。

### ConnManager

```go
cm := server.ConnManager()

cm.Count()      // int64 — 当前连接总数
cm.Get(id)      // 按 ID 获取连接，不存在返回 nil
cm.Range(fn)    // 遍历连接快照，fn 返回 false 提前停止
```

### Client

```go
client := ws.NewClient(url, opts...)   // 创建，默认 JSON

client.Connect(ctx)                    // 建立连接
client.Close()                         // 关闭客户端（幂等）
client.Send(req)                       // 发送请求，不等响应

client.OnState(fn)                     // 状态回调 fn(State)
client.OnError(fn)                     // 错误回调 fn(error)
client.OnReceive(fn)                   // 响应回调 fn(*Response)

client.State()                         // 当前状态
```

发送语义：

- `Send` 只做编码并入队（内部缓冲 512 条），不等待服务端响应。
- 未连接时返回 `ErrInvalidState`；已关闭时返回 `ErrClientClosed`。
- `req.ID` 为空时自动生成 21 位随机 ID，不修改调用方传入的结构体。
- 目前 `OnState` 在进入 `Connected` / `Reconnecting` / `Disconnected` 时触发。

### State 状态机

| 状态 | 说明 |
|---|---|
| `StateInit` | 创建后尚未连接 |
| `StateConnecting` | 连接中 |
| `StateConnected` | 已连接 |
| `StateReconnecting` | 自动重连中 |
| `StateFailed` | 连接失败或重连次数耗尽 |
| `StateDisconnected` | 调用 `Close()` 后 |

常量：`StateInit` / `StateConnecting` / `StateConnected` / `StateReconnecting` / `StateFailed` / `StateDisconnected`。

### 错误

| 错误 | 端 | 说明 |
|---|---|---|
| `ErrSendFull` | 服务端 | 发送缓冲区满（背压） |
| `ErrClientClosed` | 客户端 | Client 已关闭后调用 Send，或等待入队时被关闭 |
| `ErrConnectionLost` | 客户端 | 连接丢失时上报；重连超过最大次数时也会上报 |
| `ErrInvalidState` | 客户端 | 当前状态不允许 Send（未连接） |

## 配置选项

### 服务端

| Option | 默认值 | 说明 |
|---|---|---|
| `WithServerCodec(name)` | `"json"` | 编解码器，支持 `"json"` / `"msgpack"` / `"protobuf"` |
| `WithServerMaxConns(n)` | `100000` | 最大连接数，`0` 不限制 |
| `WithServerSendBufferSize(n)` | `1000` | 每连接发送缓冲区大小 |
| `WithServerHeartbeat(d)` | `30s` | 心跳配置值，实际 Ping 间隔为 `d/2`（最小 1s），连续 3 次失败断开；`0` 禁用 |
| `WithServerMaxMessageSize(n)` | `0` | 单条消息最大字节数，超出返回 413；`0` 不限制 |
| `WithServerAllowedOrigins(fn)` | 允许所有 | Origin 校验函数，返回 `false` 时握手返回 403 |

### 客户端

| Option | 默认值 | 说明 |
|---|---|---|
| `WithClientCodec(name)` | `"json"` | 编解码器，支持 `"json"` / `"msgpack"` / `"protobuf"` |
| `WithClientDialTimeout(d)` | `30s` | 建连超时 |
| `WithClientAutoReconnect(bool)` | `false` | 是否自动重连 |
| `WithClientMaxReconnectAttempts(n)` | `10` | 最大重连次数 |
| `WithClientHeartbeat(d)` | `15s` | Ping 间隔，连续 3 次失败判定连接丢失；`0` 禁用 |

重连退避：基础 500ms，每次翻倍，上限 60s，附加随机抖动；超过最大次数后进入 `StateFailed`，并通过 `OnError` 上报 `ErrConnectionLost`。

## 消息格式

### Request（客户端 → 服务端）

```json
{
  "id": "请求ID，客户端可留空由框架生成",
  "action": "echo",
  "data": <任意 JSON 数据>
}
```

### Response（服务端 → 客户端）

```json
{
  "id": "原样返回的请求ID",
  "ts": 1716700000000,
  "action": "echo",
  "code": 200,
  "msg": "",
  "data": <任意 JSON 数据>
}
```

- `data` 字段为 `json.RawMessage`，延迟解码，按需解析。
- `code` 为 `0` 或 `200` 均表示成功，示例中统一使用 `200`；`msg` 仅在失败时填写。
- 二进制编码（MsgPack / Protobuf）下 `data` 为原始字节；Protobuf 消息定义见 `protocol/ws.proto`。

## 目录结构

```
ws/
├── ws.go                 # 顶层 Facade，re-export 类型 + 构造函数 + Option
├── ws_types.go           # Request/Response 类型 re-export
├── server/
│   ├── server.go         # Server、ConnManager、topicManager
│   ├── conn.go           # Conn 连接（readLoop/writeLoop/healthLoop）
│   └── option.go         # ServerOption
├── client/
│   ├── client.go         # Client（双层 context、自动重连）
│   └── option.go         # ClientOption
├── codec/
│   ├── codec.go          # Codec 接口、NewCodec 工厂
│   ├── json.go           # JSON 编解码器
│   ├── msgpack.go        # MsgPack 编解码器
│   └── protobuf.go       # Protobuf 编解码器
├── protocol/
│   ├── ws.proto          # Protobuf 消息定义
│   └── ws.pb.go          # protoc 生成代码
└── types/
    └── message.go        # Request/Response 结构体定义
```

## 示例

完整可运行示例见仓库 `examples/ws/` 目录：

- `examples/ws/server` — 服务端：中间件、广播、连接管理、优雅关闭
- `examples/ws/client` — 客户端：回调、基础请求、并发压测
- `examples/ws/web` — 带静态页面的综合示例：订阅发布、元数据、连接遍历
