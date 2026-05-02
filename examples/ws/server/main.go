package main

import (
	"log"
	"net/http"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

// loggingMiddleware 日志中间件
func loggingMiddleware(next ws.Handler) ws.Handler {
	return func(conn *ws.Conn, req *ws.Request) {
		start := time.Now()
		log.Printf("[日志] 请求: action=%s, id=%s", req.Action, req.ID)
		next(conn, req)
		log.Printf("[日志] 响应完成，耗时=%v", time.Since(start))
	}
}

// authMiddleware 认证中间件
func authMiddleware(next ws.Handler) ws.Handler {
	return func(conn *ws.Conn, req *ws.Request) {
		token := conn.Meta["token"]
		if token == nil {
			conn.SendError(req.ID, req.Action, 401, "unauthorized")
			return
		}
		next(conn, req)
	}
}

func main() {
	// 创建服务端
	server := ws.NewServer(
		ws.NewJSONCodec(),
		ws.WithServerMaxConns(100000),
		ws.WithServerWorkerPoolSize(32),
		ws.WithServerHeartbeat(30*time.Second, 60*time.Second),
		ws.WithServerRateLimit(100000),
	)

	// 注册中间件（按注册顺序执行）
	server.Use(loggingMiddleware)
	server.Use(authMiddleware)

	// 注册处理器
	server.Handle("echo", func(conn *ws.Conn, req *ws.Request) {
		conn.SendOK(req.ID, req.Action, req.Data)
	})

	server.Handle("ping", func(conn *ws.Conn, req *ws.Request) {
		conn.SendOK(req.ID, req.Action, []byte(`{"status":"ok"}`))
	})

	server.Handle("info", func(conn *ws.Conn, req *ws.Request) {
		conn.SendError(req.ID, req.Action, 0, "pong")
	})

	server.OnConnect(func(conn *ws.Conn) {
		log.Printf("客户端连接: %s", conn.ID)
	})
	server.OnDisconnect(func(conn *ws.Conn) {
		log.Printf("客户端断开: %s", conn.ID)
	})

	// 性能指标
	metrics := server.Metrics()
	log.Printf("服务端指标: %+v", metrics)

	// 启动服务端
	mux := http.NewServeMux()
	mux.Handle("/ws", server)

	log.Printf("服务端启动 :9092")
	log.Fatal(http.ListenAndServe(":9092", mux))
}
