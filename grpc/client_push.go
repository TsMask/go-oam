package grpc

import (
	"context"
	"io"

	"github.com/tsmask/go-oam/grpc/protobuf"
	"github.com/tsmask/go-oam/grpc/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PushStream 接收服务端推送的流
type PushStream struct {
	ctx    context.Context
	client *Client
	conn   *Conn
	stream protobuf.Service_StreamClient
}

func newPushStream(ctx context.Context, endpoint string, conn *Conn) (*PushStream, error) {
	grpcConn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	stream, err := protobuf.NewServiceClient(grpcConn).Stream(ctx)
	if err != nil {
		return nil, err
	}

	return &PushStream{
		ctx:    ctx,
		conn:   conn,
		stream: stream,
	}, nil
}

// readLoop 读取服务端请求
func (s *PushStream) readLoop() {
	defer s.conn.Close()

	for {
		pbMsg, err := s.stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			if s.client.onError != nil {
				s.client.onError(err)
			}
			return
		}

		msg := &types.Message{
			ID:     pbMsg.Id,
			Action: pbMsg.Action,
			Data:   pbMsg.Data,
			Code:   pbMsg.Code,
			Msg:    pbMsg.Msg,
			Ts:     pbMsg.Ts,
		}

		// 分发请求到 handler
		s.dispatch(msg)
	}
}

// writeLoop 发送响应
func (s *PushStream) writeLoop() {
	defer s.conn.Close()

	for {
		select {
		case <-s.conn.done:
			return
		case msg := <-s.conn.sendCh:
			pbMsg := &protobuf.Message{
				Id:     msg.ID,
				Action: msg.Action,
				Data:   msg.Data,
				Code:   msg.Code,
				Msg:    msg.Msg,
				Ts:     msg.Ts,
			}
			if err := s.stream.Send(pbMsg); err != nil {
				if s.client.onError != nil {
					s.client.onError(err)
				}
				return
			}
		}
	}
}

// dispatch 消息分发
func (s *PushStream) dispatch(msg *types.Message) {
	handler, ok := s.client.handlers.Get(msg.Action)
	if !ok {
		resp := &types.Message{}
		resp.SetError(msg.ID, 404, "handler not found")
		s.conn.sendCh <- resp
		return
	}

	go func() {
		result, err := handler(s.ctx, msg.Data)
		resp := &types.Message{}
		if err != nil {
			resp.SetError(msg.ID, 500, err.Error())
		} else {
			resp.SetSuccess(msg.ID, result)
		}
		s.conn.sendCh <- resp
	}()
}
