# WebSocket 通信框架

高性能 WebSocket 通信框架，支持请求-响应模式、发布订阅、中间件和自动重连。

基于 [coder/websocket](https://github.com/coder/websocket) 实现。

## 特性

- **请求-响应** — 异步发送，按 `id` 匹配响应，不阻塞等待
- **多编码自动检测** — 同一服务端同时服务 JSON/MsgPack/Protobuf 客户端，按消息类型自动切换
- **发布订阅** — 内置 Topic 管理，`Subscribe`/`Publish`/`Broadcast` 一应俱全
- **中间件** — 洋葱模型，按注册顺序包裹 Handler
- **自动重连** — 指数退避 + 抖动，客户端内置
- **心跳保活** — 服务端/客户端均可配置，连续 3 次 Ping 失败断开
- **优雅关闭** — `Shutdown()` 标记关闭 → 拒绝新连接 → 关闭所有已有连接
- **元数据** — 每连接 `SetMeta`/`GetMeta`，线程安全
- **非阻塞发送** — 缓冲区满返回 `ErrSendFull`，不卡写循环

## 快速开始

### 服务端

```go
server := ws.NewServer(
    ws.WithServerCodec("json"),
    ws.WithServerMaxConns(10000),
    ws.WithServerHeartbeat(30*time.Second),
)

// 中间件
server.Use(func(next ws.Handler) ws.Handler {
    return func(conn *ws.Conn, req *ws.Request) {
        start := time.Now()
        next(conn, req)
        log.Printf("[MW] %s %s %v", conn.ID()[:8], req.Action, time.Since(start))
    }
})

// 注册处理器
server.Handle("echo", func(conn *ws.Conn, req *ws.Request) {
    conn.SendOK(req.ID, req.Action, req.Data)
})

// 连接回调 — r 是原始 HTTP 请求，可读取 Header/Cookie 等
server.OnConnect(func(conn *ws.Conn, r *http.Request) {
    log.Printf("连接: %s %s", conn.ID(), r.RemoteAddr)
})
server.OnDisconnect(func(conn *ws.Conn) {
    log.Printf("断开: %s", conn.ID())
})

// 挂载到 HTTP 路由
mux := http.NewServeMux()
mux.Handle("/ws", server)
log.Fatal(http.ListenAndServe(":9092", mux))

// 优雅关闭
server.Shutdown()
```

### 客户端

```go
client := ws.NewClient("ws://localhost:9092/ws",
    ws.WithClientAutoReconnect(true),
)

if err := client.Connect(context.Background()); err != nil {
    log.Fatal(err)
}
defer client.Close()

// 设置响应回调
client.OnReceive(func(resp *ws.Response) {
    log.Printf("收到响应: id=%s code=%d", resp.ID, resp.Code)
})

// 发送请求（非阻塞，不等响应）
err := client.Send(&ws.Request{
    Action: "echo",
    Data:   []byte(`"hello"`),
})
```

## 编解码

通过 `WithServerCodec(name)` / `WithClientCodec(name)` 设置，支持 `"json"`、`"msgpack"`、`"protobuf"`，默认 `"json"`。

服务端**自动检测**客户端编码：

| 客户端发送 | WebSocket 消息类型 | 服务端解码 |
| ---------- | ------------------ | ---------- |
| JSON       | Text (1)           | JSON       |
| MsgPack    | Binary (2)         | MsgPack    |
| Protobuf   | Binary (2)         | Protobuf   |

响应编码与请求一致 — 客户端发 JSON，服务端回 JSON；客户端发 MsgPack，服务端回 MsgPack。

浏览器始终发 JSON（Text 消息），Go 客户端可选用 MsgPack 或 Protobuf 提升吞吐。**同一服务端可同时服务两类客户端**。

> 当服务端配置了 `"protobuf"` 而浏览器发来 JSON 时，服务端仍能正确解码（Text 消息走 JSON 路径），并用 JSON 回复浏览器。

## 发布订阅

```go
// 服务端 — 在 Handler 中让连接订阅
server.Handle("subscribe", func(conn *ws.Conn, req *ws.Request) {
    conn.Subscribe("news", "alerts")
})

// 向 topic 发布消息
server.Publish("news", []byte(`{"title":"hello"}`))

// 条件发布
server.PublishFilter("news", data, func(c *ws.Conn) bool {
    return c.ID() != senderID
})

// 广播所有连接
server.Broadcast("notification", data)

// 条件广播
server.BroadcastFilter("notification", data, func(c *ws.Conn) bool {
    return c.ID() != senderID
})

// 查询
server.Topics()           // []string — 所有活跃 topic
server.TopicCount("news") // int — 订阅者数量
conn.Subscriptions()      // []string — 当前连接订阅的 topic
```

## 与 Gin 集成

```go
r := gin.Default()

// 在 Gin 中间件里做鉴权
r.GET("/ws", func(c *gin.Context) {
    user := c.GetString("user") // 从 JWT/session 取
    server.ServeHTTP(c.Writer, c.Request)
    // 注意：ServeHTTP 内部已升级协议，后续操作通过 Conn.SetMeta 存储
})

// 更好的方式：OnConnect 回调中通过 *http.Request 读取 Gin 上下文数据
server.OnConnect(func(conn *ws.Conn, r *http.Request) {
    // r.Header.Get("Authorization") 等均可访问
    conn.SetMeta("user", r.Header.Get("X-User"))
})
```

## API 参考

### Server

```go
server := ws.NewServer(opts...)          // 创建，默认 JSON

server.Use(middleware...)                // 注册中间件
server.Handle(action, handler)           // 注册处理器（线程安全）

server.OnConnect(fn)                     // 连接回调 fn(conn, *http.Request)
server.OnDisconnect(fn)                  // 断开回调 fn(conn)

server.Broadcast(action, data)           // 广播所有连接
server.BroadcastFilter(action, data, fn) // 条件广播
server.Publish(topic, data)              // 向 topic 发布
server.PublishFilter(topic, data, fn)    // 条件发布
server.Topics()                          // 活跃 topic 列表
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
conn.LastActiveTime()        // 最后活跃时间
conn.CodecName()             // 当前响应编码器名称

conn.Send(id, action, code, data)    // 发送响应
conn.SendOK(id, action, data)        // 发送成功响应 (code=200)
conn.SendError(id, action, code, msg)// 发送错误响应

conn.SetMeta(key, val)               // 设置元数据
conn.GetMeta(key)                    // 获取元数据

conn.Subscribe(topics...)            // 订阅
conn.Unsubscribe(topics...)          // 取消订阅
conn.Subscriptions()                 // 已订阅 topic 列表

conn.Close()                         // 关闭连接
```

### ConnManager

```go
cm := server.ConnManager()

cm.Count()            // 当前连接总数
cm.Get(id)            // 按 ID 获取连接
cm.Range(fn)          // 遍历，fn 返回 false 停止
```

### Client

```go
client := ws.NewClient(url, opts...)   // 创建，默认 JSON

client.Connect(ctx)                    // 建立连接
client.Close()                         // 关闭客户端

client.Send(req)                       // 发送请求（非阻塞）

client.OnState(fn)                     // 状态变更 fn(State)
client.OnError(fn)                     // 错误 fn(error)
client.OnReceive(fn)                   // 响应 fn(*Response)

client.State()                         // 当前状态
```

### State 状态机

```
Init → Connecting → Connected ⇄ Reconnecting → Failed
                        ↓
                   Disconnected (Close)
```

## 配置选项

### 服务端

| Option                         | 默认值   | 说明                                                 |
| ------------------------------ | -------- | ---------------------------------------------------- |
| `WithServerCodec(name)`        | `"json"` | 编解码器，支持 `"json"` / `"msgpack"` / `"protobuf"` |
| `WithServerMaxConns(n)`        | `100000` | 最大连接数，0 不限制                                 |
| `WithServerSendBufferSize(n)`  | `1000`   | 每连接发送缓冲区大小                                 |
| `WithServerHeartbeat(d)`       | `30s`    | 心跳间隔，0 禁用                                     |
| `WithServerMaxMessageSize(n)`  | `0`      | 最大消息字节数，0 不限制                             |
| `WithServerAllowedOrigins(fn)` | 允许所有 | 跨域来源验证函数                                     |

### 客户端

| Option                              | 默认值   | 说明             |
| ----------------------------------- | -------- | ---------------- |
| `WithClientCodec(name)`             | `"json"` | 编解码器         |
| `WithClientDialTimeout(d)`          | `30s`    | 连接超时         |
| `WithClientAutoReconnect(bool)`     | `false`  | 自动重连         |
| `WithClientMaxReconnectAttempts(n)` | `10`     | 最大重连次数     |
| `WithClientHeartbeat(d)`            | `15s`    | 心跳间隔，0 禁用 |

## 消息格式

### Request（客户端 → 服务端）

```json
{
  "id": "随机ID",
  "action": "echo",
  "data": <任意数据>
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
  "data": <任意数据>
}
```

`data` 字段为 `json.RawMessage`，延迟解码，服务端/客户端按需解析。二进制数据通过 base64 字符串在 JSON 中传输。

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
├── protocol/             # Protobuf 定义（ws.proto → ws.pb.go）
└── types/
    └── message.go        # Request/Response 结构体定义
```
