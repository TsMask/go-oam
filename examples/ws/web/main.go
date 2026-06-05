package main

import (
	"encoding/json"
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

// go run main.go
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
		ws.WithServerCodec("protobuf"),         // 编解码器
		ws.WithServerMaxConns(1000),            // 最大连接数
		ws.WithServerSendBufferSize(2000),      // 发送缓冲区
		ws.WithServerHeartbeat(30*time.Second), // 心跳 30s
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
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: req.Data})
	})

	// ping — 健康检查
	server.Handle("ping", func(conn *ws.Conn, req *ws.Request) {
		data, _ := json.Marshal(map[string]string{
			"status": "ok",
		})
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: data})
	})

	// info — 连接信息（ConnManager.Count + Conn.GetMeta）
	server.Handle("info", func(conn *ws.Conn, req *ws.Request) {
		name, _ := conn.GetMeta("name")
		addr, _ := conn.GetMeta("remote_addr")
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: []byte(fmt.Sprintf(
			`{"id":"%s","name":"%v","addr":"%v","connections":%d}`,
			conn.ID(), name, addr, server.ConnManager().Count(),
		))})
	})

	// set_name — 设置连接元数据（SetMeta）
	server.Handle("set_name", func(conn *ws.Conn, req *ws.Request) {
		name := strings.Trim(string(req.Data), `"`)
		conn.SetMeta("name", name)
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: []byte(fmt.Sprintf(`{"name":"%s"}`, name))})
	})

	// broadcast — 全部广播（Broadcast）
	server.Handle("broadcast", func(conn *ws.Conn, req *ws.Request) {
		server.Broadcast(&ws.Response{ID: req.ID, Action: "notification", Code: 200, Data: req.Data})
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: []byte(`{"sent":true}`)})
	})

	// targeted — 条件广播，排除发送者（BroadcastFilter）
	server.Handle("targeted", func(conn *ws.Conn, req *ws.Request) {
		server.BroadcastFilter(
			&ws.Response{ID: req.ID, Action: "targeted_msg", Code: 200, Data: req.Data},
			func(c *ws.Conn) bool {
				return c.ID() != conn.ID()
			},
		)
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: []byte(`{"sent":true,"exclude_self":true}`)})
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
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: []byte(
			fmt.Sprintf(`{"count":%d,"clients":[%s]}`, len(items), strings.Join(items, ",")),
		)})
	})

	// get_conn — 按 ID 查找连接（ConnManager.Get）
	server.Handle("get_conn", func(conn *ws.Conn, req *ws.Request) {
		targetID := strings.Trim(string(req.Data), `"`)
		target := server.ConnManager().Get(targetID)
		if target == nil {
			conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 404, Data: []byte("连接不存在")})
			return
		}
		name, _ := target.GetMeta("name")
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: []byte(fmt.Sprintf(
			`{"id":"%s","name":"%v","last_active":"%s"}`,
			target.ID(), name, target.LastActiveTime().Format(time.RFC3339),
		))})
	})

	// subscribe — 订阅 topic（Conn.Subscribe）
	server.Handle("subscribe", func(conn *ws.Conn, req *ws.Request) {
		var body struct {
			Topics []string `json:"topics"`
		}
		if err := json.Unmarshal(req.Data, &body); err != nil || len(body.Topics) == 0 {
			conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 400, Data: []byte("invalid topics")})
			return
		}
		conn.Subscribe(body.Topics...)
		topics := conn.Subscriptions()
		data, _ := json.Marshal(map[string]any{"subscribed": body.Topics, "all": topics})
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: data})
	})

	// unsubscribe — 取消订阅 topic（Conn.Unsubscribe）
	server.Handle("unsubscribe", func(conn *ws.Conn, req *ws.Request) {
		var body struct {
			Topics []string `json:"topics"`
		}
		if err := json.Unmarshal(req.Data, &body); err != nil || len(body.Topics) == 0 {
			conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 400, Data: []byte("invalid topics")})
			return
		}
		conn.Unsubscribe(body.Topics...)
		topics := conn.Subscriptions()
		data, _ := json.Marshal(map[string]any{"unsubscribed": body.Topics, "all": topics})
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: data})
	})

	// publish — 向 topic 发布消息（Server.Publish）
	server.Handle("publish", func(conn *ws.Conn, req *ws.Request) {
		var body struct {
			Topic string `json:"topic"`
		}
		if err := json.Unmarshal(req.Data, &body); err != nil || body.Topic == "" {
			conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 400, Data: []byte("invalid topic")})
			return
		}
		count := server.TopicCount(body.Topic)
		server.Publish(body.Topic, &ws.Response{Action: body.Topic, Code: 200, Data: req.Data})
		data, _ := json.Marshal(map[string]any{"topic": body.Topic, "delivered": count})
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: data})
	})

	// topics — 查看所有 topic 和当前订阅（Server.Topics + Conn.Subscriptions）
	server.Handle("topics", func(conn *ws.Conn, req *ws.Request) {
		all := server.Topics()
		mine := conn.Subscriptions()
		result := map[string]any{
			"all_topics": all,
			"my_topics":  mine,
		}
		for _, t := range all {
			result[t+"_count"] = server.TopicCount(t)
		}
		data, _ := json.Marshal(result)
		conn.SendResp(&ws.Response{ID: req.ID, Action: req.Action, Code: 200, Data: data})
	})

	// 连接回调 — 可通过 r 访问 HTTP 请求信息（Header/Cookie/Gin Context）
	server.OnConnect(func(conn *ws.Conn, r *http.Request) {
		conn.SetMeta("name", conn.ID()[:8])
		conn.SetMeta("remote_addr", r.RemoteAddr)
		conn.SetMeta("user_agent", r.UserAgent())
		log.Printf("[CONNECT] %s %s", conn.ID(), r.RemoteAddr)
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
