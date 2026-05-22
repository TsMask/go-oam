package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", fs)

	go func() {
		fmt.Printf("HTTP 静态服务: http://localhost:%s\n", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// 创建服务端 — 使用全部配置项
	server := ws.NewServer(
		ws.NewJSONCodec(),
		ws.WithServerMaxConns(1000),            // 最大连接数
		ws.WithServerSendBufferSize(2000),      // 发送缓冲区
		ws.WithServerWorkerPoolSize(4),         // Worker 池
		ws.WithServerJobQueueSize(50),          // 任务队列
		ws.WithServerHeartbeat(30*time.Second), // 心跳 30s
		ws.WithServerRateLimit(10000),          // 限流
		ws.WithServerAllowedOrigins(func(origin string) bool {
			return true // 允许所有来源
		}),
		ws.WithServerMaxMessageSize(4096), // 最大消息 4KB
	)

	// 中间件 — 日志
	server.Use(func(next ws.Handler) ws.Handler {
		return func(conn *ws.Conn, req *ws.Request) {
			start := time.Now()
			next(conn, req)
			log.Printf("[MW] %s %s %v", conn.ID()[:8], req.Action, time.Since(start))
		}
	})

	// echo — 原样返回
	server.Handle("echo", func(conn *ws.Conn, req *ws.Request) {
		conn.SendOK(req.ID, req.Action, req.Data)
	})

	// ping — 健康检查
	server.Handle("ping", func(conn *ws.Conn, req *ws.Request) {
		conn.SendOK(req.ID, req.Action, []byte(`{"status":"ok"}`))
	})

	// info — 连接信息（ConnManager.Count + Conn.GetMeta）
	server.Handle("info", func(conn *ws.Conn, req *ws.Request) {
		name, _ := conn.GetMeta("name")
		addr, _ := conn.GetMeta("remote_addr")
		conn.SendOK(req.ID, req.Action, []byte(fmt.Sprintf(
			`{"id":"%s","name":"%v","addr":"%v","connections":%d}`,
			conn.ID(), name, addr, server.ConnManager().Count(),
		)))
	})

	// set_name — 设置连接元数据（SetMeta）
	server.Handle("set_name", func(conn *ws.Conn, req *ws.Request) {
		name := strings.Trim(string(req.Data), `"`)
		conn.SetMeta("name", name)
		conn.SendOK(req.ID, req.Action, []byte(fmt.Sprintf(`{"name":"%s"}`, name)))
	})

	// broadcast — 全部广播（Broadcast）
	server.Handle("broadcast", func(conn *ws.Conn, req *ws.Request) {
		server.Broadcast("notification", req.Data)
		conn.SendOK(req.ID, req.Action, []byte(`{"sent":true}`))
	})

	// targeted — 条件广播，排除发送者（BroadcastFilter）
	server.Handle("targeted", func(conn *ws.Conn, req *ws.Request) {
		server.BroadcastFilter("targeted_msg", req.Data, func(c *ws.Conn) bool {
			return c.ID() != conn.ID()
		})
		conn.SendOK(req.ID, req.Action, []byte(`{"sent":true,"exclude_self":true}`))
	})

	// list — 遍历连接（ConnManager.Range + Conn.ID + Conn.LastActiveTime）
	server.Handle("list", func(conn *ws.Conn, req *ws.Request) {
		var items []string
		server.ConnManager().Range(func(c *ws.Conn) bool {
			name, _ := c.GetMeta("name")
			items = append(items, fmt.Sprintf(`{"id":"%s","name":"%v","last_active":"%s"}`,
				c.ID(), name, c.LastActiveTime().Format(time.RFC3339)))
			return true
		})
		conn.SendOK(req.ID, req.Action, []byte(
			fmt.Sprintf(`{"count":%d,"clients":[%s]}`, len(items), strings.Join(items, ",")),
		))
	})

	// get_conn — 按 ID 查找连接（ConnManager.Get）
	server.Handle("get_conn", func(conn *ws.Conn, req *ws.Request) {
		targetID := strings.Trim(string(req.Data), `"`)
		target := server.ConnManager().Get(targetID)
		if target == nil {
			conn.SendError(req.ID, req.Action, 404, "连接不存在")
			return
		}
		name, _ := target.GetMeta("name")
		conn.SendOK(req.ID, req.Action, []byte(fmt.Sprintf(
			`{"id":"%s","name":"%v","last_active":"%s"}`,
			target.ID(), name, target.LastActiveTime().Format(time.RFC3339),
		)))
	})

	// 连接回调
	server.OnConnect(func(conn *ws.Conn) {
		conn.SetMeta("name", conn.ID()[:8])
		log.Printf("[CONNECT] %s", conn.ID())
	})
	server.OnDisconnect(func(conn *ws.Conn) {
		log.Printf("[DISCONNECT] %s", conn.ID())
	})

	// 挂载 WebSocket 到独立 HTTP 端口
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

	wsAddr := ":9092"
	fmt.Printf("WebSocket 服务端: %s\n", wsAddr)
	log.Fatal(http.ListenAndServe(wsAddr, mux))
}
