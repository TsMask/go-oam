package main

import (
	"context"
	"log"

	"github.com/tsmask/go-oam/nms"
)

func main() {
	ctx := context.Background()

	// 创建 NE 客户端
	cli := nms.NewClient(
		nms.WithAddr("localhost:50051"),
		nms.WithNEID("ne-001"),
		nms.WithNEType("router"),
	)

	// 设置回调
	cli.SetOnConnected(func() {
		log.Printf("[NE] Connected to NMS")
	})

	cli.SetOnDisconnected(func() {
		log.Printf("[NE] Disconnected from NMS")
	})

	// 连接 NMS
	if err := cli.Connect(ctx); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer cli.Disconnect(ctx)

	log.Printf("[NE] Demo completed - connected to NMS")
}