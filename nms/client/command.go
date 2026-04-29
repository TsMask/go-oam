package client

import (
	"context"
	"io"

	"google.golang.org/grpc"

	pb "github.com/tsmask/go-oam/nms/proto"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// CommandClient 命令控制客户端
// ============================================

type CommandClient struct {
	client *Client
}

// newCommandClient 创建命令客户端
func newCommandClient(c *Client) *CommandClient {
	return &CommandClient{client: c}
}

// Execute 执行单条命令
func (c *CommandClient) Execute(ctx context.Context, command string, commandType string, opts ...CommandOption) (*types.CommandResponse, error) {
	cli := c.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	req := &pb.CommandRequest{
		NeId:       c.client.opts.NEID,
		Command:    command,
		CommandType: commandType,
		Timeout:    30,
	}

	// 应用选项
	for _, opt := range opts {
		opt(req)
	}

	resp, err := cli.ExecuteCommand(ctx, req)
	if err != nil {
		return nil, err
	}

	return &types.CommandResponse{
		Code:     resp.Code,
		Message:  resp.Message,
		Result:   resp.Result,
		ExecTime: resp.ExecTime,
	}, nil
}

// BatchExecute 批量执行命令
func (c *CommandClient) BatchExecute(ctx context.Context, neIDs []string, command string, commandType string, opts ...BatchCommandOption) (<-chan *types.BatchCommandResponse, error) {
	cli := c.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	req := &pb.BatchCommandRequest{
		NeIds:       neIDs,
		SessionId:  "",
		Command:    command,
		CommandType: commandType,
		Timeout:    30,
	}

	// 应用选项
	for _, opt := range opts {
		opt(req)
	}

	// 创建流
	stream, err := cli.BatchExecuteCommand(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}

	// 接收响应流
	ch := make(chan *types.BatchCommandResponse, 1)
	go func() {
		defer close(ch)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			ch <- &types.BatchCommandResponse{
				NEID:     resp.NeId,
				Code:     resp.Code,
				Message:  resp.Message,
				Result:   resp.Result,
				ExecTime: resp.ExecTime,
			}
		}
	}()

	return ch, nil
}

// ============================================
// CommandOption 命令选项
// ============================================

type CommandOption func(*pb.CommandRequest)

func WithTimeout(timeout int32) CommandOption {
	return func(req *pb.CommandRequest) {
		req.Timeout = timeout
	}
}

func WithParams(params map[string]string) CommandOption {
	return func(req *pb.CommandRequest) {
		req.Params = params
	}
}

type BatchCommandOption func(*pb.BatchCommandRequest)

func WithBatchTimeout(timeout int32) BatchCommandOption {
	return func(req *pb.BatchCommandRequest) {
		req.Timeout = timeout
	}
}

func WithBatchParams(params map[string]string) BatchCommandOption {
	return func(req *pb.BatchCommandRequest) {
		req.Params = params
	}
}

// ============================================
// 会话上下文（内部使用）
// ============================================

func (c *CommandClient) ExecuteWithSession(ctx context.Context, sessionID string, command string, commandType string, timeout int32) (*types.CommandResponse, error) {
	cli := c.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	req := &pb.CommandRequest{
		NeId:       c.client.opts.NEID,
		SessionId:  sessionID,
		Command:    command,
		CommandType: commandType,
		Timeout:    timeout,
	}

	resp, err := cli.ExecuteCommand(ctx, req)
	if err != nil {
		return nil, err
	}

	return &types.CommandResponse{
		Code:     resp.Code,
		Message:  resp.Message,
		Result:   resp.Result,
		ExecTime: resp.ExecTime,
	}, nil
}