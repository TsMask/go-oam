package client

import (
	"context"
	"time"

	pb "github.com/tsmask/go-oam/nms/proto"
	"github.com/tsmask/go-oam/nms/types"
)

// ============================================
// MetricsClient 性能数据客户端（NE 上报性能到 NMS）
// ============================================

type MetricsClient struct {
	client *Client
}

// newMetricsClient 创建性能客户端
func newMetricsClient(c *Client) *MetricsClient {
	return &MetricsClient{client: c}
}

// Send 发送性能数据
func (m *MetricsClient) Send(ctx context.Context, metricsType string, items []*types.MetricItem) (*types.MetricsResponse, error) {
	cli := m.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	// 创建流
	stream, err := cli.ReportMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// 发送性能数据
	err = stream.Send(&pb.MetricsRequest{
		NeId:        m.client.opts.NEID,
		MetricsType: metricsType,
		Timestamp:   time.Now().Unix(),
		Metrics:     convertMetricItemsToProto(items),
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

	return &types.MetricsResponse{
		NEID:    resp.NeId,
		Ack:     resp.Ack,
		Message: resp.Message,
	}, nil
}

// SendStream 发送性能数据流（bidirectional streaming）
func (m *MetricsClient) SendStream(ctx context.Context) (*MetricsStream, error) {
	cli := m.client.getClient()
	if cli == nil {
		return nil, ErrNotConnected
	}

	stream, err := cli.ReportMetrics(ctx)
	if err != nil {
		return nil, err
	}

	return &MetricsStream{
		ctx:    ctx,
		stream: stream,
		neID:   m.client.opts.NEID,
	}, nil
}

// MetricsStream 性能数据流
type MetricsStream struct {
	ctx         context.Context
	stream      pb.NMSService_ReportMetricsClient
	neID        string
	metricsType string
}

// SetMetricsType 设置指标类型
func (s *MetricsStream) SetMetricsType(t string) {
	s.metricsType = t
}

// Send 发送性能数据
func (s *MetricsStream) Send(items []*types.MetricItem) error {
	return s.stream.Send(&pb.MetricsRequest{
		NeId:        s.neID,
		MetricsType: s.metricsType,
		Timestamp:   time.Now().Unix(),
		Metrics:     convertMetricItemsToProto(items),
	})
}

// Recv 接收响应
func (s *MetricsStream) Recv() (*types.MetricsResponse, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		return nil, err
	}
	return &types.MetricsResponse{
		NEID:    resp.NeId,
		Ack:     resp.Ack,
		Message: resp.Message,
	}, nil
}

// Close 关闭流
func (s *MetricsStream) Close() error {
	return s.stream.CloseSend()
}

// ============================================
// 辅助函数
// ============================================

func convertMetricItemsToProto(items []*types.MetricItem) []*pb.MetricItem {
	result := make([]*pb.MetricItem, len(items))
	for i, item := range items {
		result[i] = &pb.MetricItem{
			Name:   item.Name,
			Value:  item.Value,
			Unit:   item.Unit,
			Labels: item.Labels,
		}
	}
	return result
}

func convertMetricItemsFromProto(pbMetrics []*pb.MetricItem) []*types.MetricItem {
	result := make([]*types.MetricItem, len(pbMetrics))
	for i, m := range pbMetrics {
		result[i] = &types.MetricItem{
			Name:   m.Name,
			Value:  m.Value,
			Unit:   m.Unit,
			Labels: m.Labels,
		}
	}
	return result
}