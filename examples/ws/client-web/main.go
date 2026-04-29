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
	"github.com/tsmask/go-oam/ws/codec"
	"github.com/tsmask/go-oam/ws/types"
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
		fmt.Printf("📂 HTTP 服务器启动: http://localhost:%s\n", port)
		fmt.Printf("📁 静态文件目录: %s\n", dir)
		fmt.Printf("🌐 访问地址: http://localhost:%s/index.html\n", port)
		fmt.Println()
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	server := ws.NewServer(":9092", codec.NewJSONCodec(),
		ws.WithServerMaxConns(100000),
		ws.WithServerWorkerPoolSize(32),
		ws.WithServerHeartbeatInterval(30*time.Second),
		ws.WithServerHeartbeatTimeout(60*time.Second),
		ws.WithServerRateLimit(100000),
	)

	server.Handle("echo", func(conn *ws.Conn, req *types.Request) {
		conn.SendResp(&types.Response{
			Code: 0,
			Data: req.Data,
		})
	})

	server.Handle("ping", func(conn *ws.Conn, req *types.Request) {
		conn.SendResp(&types.Response{
			Code: 0,
			Msg:  "pong",
			Data: []byte(`{"status":"ok"}`),
		})
	})

	server.Handle("info", func(conn *ws.Conn, req *types.Request) {
		conn.SendResp(&types.Response{
			Code: 0,
			Msg:  "success",
			Data: []byte(fmt.Sprintf(`{"id":"%s","connections":%d}`, conn.ID, server.ConnManager().Count())),
		})
	})

	server.OnConnect = func(conn *ws.Conn) {
		log.Printf("客户端连接: %s", conn.ID)
	}
	server.OnDisconnect = func(conn *ws.Conn) {
		log.Printf("客户端断开: %s", conn.ID)
	}

	metrics := server.Metrics()
	log.Printf("服务端指标: %+v", metrics)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("收到信号，关闭中...")
		server.Stop()
	}()

	fmt.Println("🔌 WebSocket 服务端启动 :9092")
	fmt.Println("📊 支持的消息类型: echo, ping, info")
	fmt.Println()
	log.Printf("服务端启动 :9092")
	log.Fatal(server.Start())
}
