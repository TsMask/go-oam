package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
)

var client *ws.ClientConn
var wsUrl = "ws://127.0.0.1:33030/ws"

// WS模块调试-客户端
// go run ./examples/ws_client
//
//	curl -X POST http://127.0.0.1:8081/send?msgType=text \
//	  -H "Content-Type: application/json" \
//	  -d '{"uuid": "xxx","type": "net","data": {"port": 4523,"name": "", "pid": 0}}'
func main() {
	stopSignal()

	// 1. 建立ws连接
	go startWS()

	// 2. 启动HTTP服务
	startHTTP()
}

func startWS() {
	// 建立ws连接
	client = &ws.ClientConn{Url: wsUrl}
	if err := client.Connect(); err != nil {
		fmt.Println("ws connect fail", err)
		return
	}
	fmt.Println("WS connected to", wsUrl)

	go client.WriteListen(func(err error) {
		fmt.Println("WriteListen error", err)
	})
	go client.ReadListen(func(err error) {
		fmt.Println("ReadListen error", err)
	}, func(_ *ws.ClientConn, messageType int, res *protocol.Response) {
		fmt.Printf("Recv %d message: %+v\n", messageType, res)
	})
	for range client.CloseSignal() {
		fmt.Println("-- WS closed")
		break
	}
}

func startHTTP() {
	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		msgType := r.URL.Query().Get("msgType")
		messageType := websocket.TextMessage
		if msgType == "binary" {
			messageType = websocket.BinaryMessage
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}

		var payload struct {
			UUID string `json:"uuid"`
			Type string `json:"type"`
			Data any    `json:"data"`
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if client == nil {
			http.Error(w, "WS client not connected", http.StatusInternalServerError)
			return
		}

		fmt.Printf("\nSending %d message: %+v\n", messageType, payload)
		client.SendReqJSON(messageType, payload.UUID, payload.Type, payload.Data)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	fmt.Println("HTTP server listening on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Println("HTTP server error:", err)
	}
}

// stopSignal 监听退出信号
func stopSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh // 等待退出信号
		if client != nil {
			client.Close()
		}
		fmt.Println("\nStop Service... OK")
		os.Exit(0)
	}()
}
