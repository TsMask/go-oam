package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

func loggingMiddleware(next ws.Handler) ws.Handler {
	return func(conn *ws.Conn, req *ws.Request) {
		start := time.Now()
		log.Printf("[日志] 请求: action=%s, id=%s", req.Action, req.ID)
		next(conn, req)
		log.Printf("[日志] 响应完成，耗时=%v", time.Since(start))
	}
}

func authMiddleware(next ws.Handler) ws.Handler {
	return func(conn *ws.Conn, req *ws.Request) {
		token, _ := conn.GetMeta("token")
		if token == nil {
			conn.SendError(req.ID, req.Action, 401, "unauthorized")
			return
		}
		next(conn, req)
	}
}

func main() {
	// 创建服务端 — 使用全部配置项
	server := ws.NewServer(
		ws.WithServerMaxConns(100000),          // 最大连接数
		ws.WithServerSendBufferSize(2000),      // 每连接发送缓冲区
		ws.WithServerHeartbeat(30*time.Second), // 心跳间隔
		ws.WithServerAllowedOrigins(func(origin string) bool { // 来源校验
			return true
		}),
		ws.WithServerMaxMessageSize(4096), // 最大消息大小
	)

	// 注册中间件
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
		conn.SendOK(req.ID, req.Action, []byte(
			fmt.Sprintf(`{"id":"%s","connections":%d}`, conn.ID(), server.ConnManager().Count()),
		))
	})

	// 广播处理器
	server.Handle("broadcast", func(conn *ws.Conn, req *ws.Request) {
		server.Broadcast("notification", req.Data)
		conn.SendOK(req.ID, req.Action, []byte(`{"sent":true}`))
	})

	// 条件广播：只发给指定连接
	server.Handle("targeted", func(conn *ws.Conn, req *ws.Request) {
		server.BroadcastFilter("targeted_msg", req.Data, func(c *ws.Conn) bool {
			return c.ID() != conn.ID() // 不发给自己
		})
		conn.SendOK(req.ID, req.Action, []byte(`{"sent":true}`))
	})

	// 遍历连接
	server.Handle("list", func(conn *ws.Conn, req *ws.Request) {
		var ids []string
		server.ConnManager().Range(func(c *ws.Conn) bool {
			ids = append(ids, c.ID())
			return true
		})
		conn.SendOK(req.ID, req.Action, []byte(
			fmt.Sprintf(`{"count":%d,"ids":"%v"}`, len(ids), ids),
		))
	})

	// 连接回调 — 通过 r 访问 HTTP 请求信息
	server.OnConnect(func(conn *ws.Conn, r *http.Request) {
		conn.SetMeta("token", conn.ID()) // 设置元数据用于认证
		conn.SetMeta("remote_addr", r.RemoteAddr)
		conn.SetMeta("connected_at", conn.LastActiveTime().Format(time.RFC3339))
		log.Printf("客户端连接: %s, 远程地址: %s", conn.ID(), r.RemoteAddr)
	})
	server.OnDisconnect(func(conn *ws.Conn) {
		log.Printf("客户端断开: %s", conn.ID())
	})

	// 挂载到用户自建的 HTTP 服务
	mux := http.NewServeMux()
	mux.Handle("/ws", server)

	// 优雅关闭
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("收到信号，关闭中...")
		server.Shutdown()
		os.Exit(0)
	}()

	addr := ":9092"
	log.Printf("服务端启动 %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
