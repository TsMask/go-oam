# WebSocket 通信框架

高性能 WebSocket 通信框架，支持请求-响应模式、中间件、自动重连和健康检查。

基于 [coder/websocket](https://github.com/coder/websocket) 实现。

## 特性

- 请求-响应模式：同步/异步请求，自动匹配响应
- 中间件：洋葱模型，类似 Gin
- 自动重连：指数退避 + 抖动
- 限流：原子令牌桶（服务端）
- 健康检查：心跳 Ping/Pong，连续 3 次失败后关闭
- 异步分发：WorkerPool + goroutine 降级
- 优雅关闭：Shutdown 关闭所有连接
- 多编码：JSON / MessagePack / Protobuf

## 快速开始

### 服务端

```go
server := ws.NewServer(":9092",
    ws.NewJSONCodec(),
    ws.WithServerMaxConns(100000),
    ws.WithServerHeartbeat(30*time.Second),
)

server.Use(loggingMiddleware)

server.Handle("echo", func(conn *ws.Conn, req *ws.Request) {
    conn.SendOK(req.ID, req.Action, req.Data)
})

server.OnConnect(func(conn *ws.Conn) {
    log.Printf("连接: %s", conn.ID())
})

mux := http.NewServeMux()
mux.Handle("/ws", server)
log.Fatal(http.ListenAndServe(":9092", mux))
```

### 客户端

```go
client := ws.NewClient("ws://localhost:9092",
    ws.NewJSONCodec(),
    ws.WithClientAutoReconnect(true),
)

if err := client.Connect(context.Background()); err != nil {
    log.Fatal(err)
}
defer client.Close()

resp, err := client.Send(context.Background(), &ws.Request{
    Action: "echo",
    Data:   []byte("hello"),
})
```

## Server API

```go
server := ws.NewServer(addr, codec, opts...)

server.Start()                // 阻塞启动
server.Shutdown(ctx)          // 优雅关闭
server.Stop()                 // 强制关闭

server.Use(middleware...)
server.Handle(action, handler)

server.OnConnect(fn)
server.OnDisconnect(fn)

server.Broadcast(action, data)
server.BroadcastFilter(action, data, filterFn)

server.ConnManager()          // 连接管理器
server.Addr()
server.Codec()
```

## Conn API

```go
conn.ID()
conn.LastActiveTime()

conn.Send(id, action, code, data)
conn.SendOK(id, action, data)
conn.SendError(id, action, code, msg)
conn.SendResp(resp)

conn.SetMeta(key, val)
conn.GetMeta(key)

conn.Close()
```

## Client API

```go
client := ws.NewClient(url, codec, opts...)

client.Connect(ctx)
client.Close()

client.Send(ctx, req)                // 同步
client.SendAsync(ctx, req, cb)       // 异步
client.SendWithTimeout(req, d)       // 带超时

client.OnState(fn)
client.OnError(fn)
client.OnReceive(fn)

client.State()
```

## 配置选项

### Server

| Option | 默认值 |
|--------|--------|
| `WithServerMaxConns` | 100000 |
| `WithServerSendBufferSize` | 1000 |
| `WithServerWorkerPoolSize` | NumCPU*2 |
| `WithServerJobQueueSize` | 自动计算 |
| `WithServerHeartbeat` | 30s |
| `WithServerRateLimit` | 100000 |
| `WithServerMaxMessageSize` | 不限制 |
| `WithServerAllowedOrigins` | 允许所有 |

### Client

| Option | 默认值 |
|--------|--------|
| `WithClientSendBufferSize` | 1000 |
| `WithClientDialTimeout` | 30s |
| `WithClientRequestTimeout` | 60s |
| `WithClientMaxPendingRequests` | 10000 |
| `WithClientAutoReconnect` | false |
| `WithClientMaxReconnectAttempts` | 10 |

## 目录结构

```
ws/
├── conn.go              # 服务端连接（readPump/writePump/healthLoop）
├── server.go            # 服务端（ConnManager + Handler + WorkerPool + 限流）
├── client.go            # 客户端（请求-响应 + 自动重连）
├── option.go            # 配置选项
├── errors.go            # 错误定义
├── ws_types.go          # 类型 re-export
├── ws_codec.go          # 编解码器工厂
├── codec/               # 编解码器（JSON / MsgPack / Protobuf）
├── protocol/            # Protobuf 定义
└── types/               # Request/Response 结构
```
