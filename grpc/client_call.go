package grpc

import (
	"context"
	"io"

	"github.com/tsmask/go-oam/grpc/protobuf"
	"github.com/tsmask/go-oam/grpc/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CallStream 客户端调用服务端的流
type CallStream struct {
	ctx    context.Context
	client *Client
	conn   *Conn
	stream protobuf.Service_StreamClient
}

func newCallStream(ctx context.Context, endpoint string, conn *Conn) (*CallStream, error) {
	grpcConn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	stream, err := protobuf.NewServiceClient(grpcConn).Stream(ctx)
	if err != nil {
		return nil, err
	}

	return &CallStream{
		ctx:    ctx,
		conn:   conn,
		stream: stream,
	}, nil
}

// readLoop 读取响应
func (s *CallStream) readLoop() {
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

		// 转换为 types.Message
		msg := &types.Message{
			ID:     pbMsg.Id,
			Action: pbMsg.Action,
			Data:   pbMsg.Data,
			Code:   pbMsg.Code,
			Msg:    pbMsg.Msg,
			Ts:     pbMsg.Ts,
		}

		// 匹配 pending 请求
		idx := int(hashString(msg.ID) & shardMask)
		s.conn.shards[idx].mu.Lock()
		ch, ok := s.conn.shards[idx].pending[msg.ID]
		s.conn.shards[idx].mu.Unlock()

		if ok && ch != nil {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

// writeLoop 发送请求
func (s *CallStream) writeLoop() {
	defer s.conn.Close()

	// 连接时发送注册消息
	regMsg := &types.Message{}
	regMsg.SetRequest(s.conn.ID, "register", nil)

	pbMsg := &protobuf.Message{
		Id:     regMsg.ID,
		Action: regMsg.Action,
		Data:   regMsg.Data,
		Code:   regMsg.Code,
		Msg:    regMsg.Msg,
		Ts:     regMsg.Ts,
	}

	if err := s.stream.Send(pbMsg); err != nil {
		if s.client.onError != nil {
			s.client.onError(err)
		}
		return
	}

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
