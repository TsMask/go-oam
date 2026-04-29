package client

import (
	"context"

	pb "github.com/tsmask/go-oam/nms/proto"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// AlarmClient 告警客户端（NE 上报告警到 NMS）
// ============================================

type AlarmClient struct {
	client *Client
}

// newAlarmClient 创建告警客户端
func newAlarmClient(c *Client) *AlarmClient {
	return &AlarmClient{client: c}
}

// Send 发送单条告警
func (a *AlarmClient) Send(ctx context.Context, alarm *types.AlarmData) (*types.AlarmResponse, error) {
	cli := a.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	// 创建流
	stream, err := cli.ReportAlarms(ctx)
	if err != nil {
		return nil, err
	}

	// 发送告警
	err = stream.Send(&pb.AlarmRequest{
		NeId: a.client.opts.NEID,
		Alarm: &pb.Alarm{
			AlarmId:     alarm.AlarmID,
			AlarmType:   alarm.AlarmType,
			Severity:    alarm.Severity,
			Name:        alarm.Name,
			Description: alarm.Description,
			Source:      alarm.Source,
			StartTime:   alarm.StartTime,
			EndTime:     alarm.EndTime,
			Params:      alarm.Params,
		},
	})
	if err != nil {
		return nil, err
	}

	// 关闭发送并接收响应
	stream.CloseSend()
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	return &types.AlarmResponse{
		NEID:    resp.NeId,
		AlarmID: resp.AlarmId,
		Ack:     resp.Ack,
		Message: resp.Message,
	}, nil
}

// SendStream 发送告警流（bidirectional streaming）
func (a *AlarmClient) SendStream(ctx context.Context) (*AlarmStream, error) {
	cli := a.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	stream, err := cli.ReportAlarms(ctx)
	if err != nil {
		return nil, err
	}

	return &AlarmStream{
		ctx:    ctx,
		stream: stream,
		neID:   a.client.opts.NEID,
	}, nil
}

// AlarmStream 告警流
type AlarmStream struct {
	ctx    context.Context
	stream pb.NMSService_ReportAlarmsClient
	neID   string
}

// Send 发送告警
func (s *AlarmStream) Send(alarm *types.AlarmData) error {
	return s.stream.Send(&pb.AlarmRequest{
		NeId: s.neID,
		Alarm: &pb.Alarm{
			AlarmId:     alarm.AlarmID,
			AlarmType:   alarm.AlarmType,
			Severity:    alarm.Severity,
			Name:        alarm.Name,
			Description: alarm.Description,
			Source:      alarm.Source,
			StartTime:   alarm.StartTime,
			EndTime:     alarm.EndTime,
			Params:      alarm.Params,
		},
	})
}

// Recv 接收响应
func (s *AlarmStream) Recv() (*types.AlarmResponse, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		return nil, err
	}

	return &types.AlarmResponse{
		NEID:    resp.NeId,
		AlarmID: resp.AlarmId,
		Ack:     resp.Ack,
		Message: resp.Message,
	}, nil
}

// Close 关闭流
func (s *AlarmStream) Close() error {
	return s.stream.CloseSend()
}

// ============================================
// 错误定义
// ============================================

var ErrNotConnected = &ClientError{Code: 1, Message: "not connected"}

type ClientError struct {
	Code    int
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}