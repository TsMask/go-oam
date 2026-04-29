package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/tsmask/go-oam/nms"
	pb "github.com/tsmask/go-oam/nms/proto"
)

func main() {
	// 创建 NMS 服务端
	srv := nms.NewServer()

	// 设置回调
	srv.SessionManager().SetOnConnect(func(ctx *nms.SessionContext) {
		log.Printf("[NMS] NE connected: %s (%s)", ctx.NEID, ctx.IP)
	})

	srv.SessionManager().SetOnDisconnect(func(ctx *nms.SessionContext) {
		log.Printf("[NMS] NE disconnected: %s", ctx.NEID)
	})

	srv.Registry().SetOnRegister(func(ne *nms.NE) {
		log.Printf("[NMS] NE registered: %s", ne.ID)
	})

	// 创建 gRPC 服务
	grpcSrv := grpc.NewServer()
	pb.RegisterNMSServiceServer(grpcSrv, srv)

	// 监听
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("[NMS] Server starting on :50051...")
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}