package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	pb "github.com/tsmask/go-oam/nms/proto"
)

// ============================================
// Client NE 客户端（连接 NMS）
// ============================================

type Client struct {
	mu      sync.RWMutex
	addr    string
	conn    *grpc.ClientConn
	client  pb.NMSServiceClient

	// 配置
	opts *ClientOptions

	// 连接状态
	connected     bool
	lastHeartbeat time.Time

	// 回调
	onConnected    func()
	onDisconnected func()
	onError        func(error)
}

// ClientOptions 客户端配置
type ClientOptions struct {
	Addr         string            // NMS 地址
	NEID         string            // NE ID
	NEType       string            // NE 类型
	IP           string            // NE IP
	Port         int32             // NE 端口
	Attrs        map[string]string // 扩展属性
	Capabilities  map[string]string // 支持的能力

	TLSConfig   credentials.TransportCredentials
	KeepAlive   time.Duration
	MaxRetry    int              // 最大重试次数
	RetryDelay  time.Duration    // 重试间隔
}

// ============================================
// 构造函数
// ============================================

func New(opts ...ClientOption) *Client {
	o := defaultClientOptions()
	for _, fn := range opts {
		fn(o)
	}

	return &Client{
		addr: o.Addr,
		opts: o,
	}
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		KeepAlive: 30 * time.Second,
		MaxRetry:  3,
		RetryDelay: 5 * time.Second,
	}
}

type ClientOption func(*ClientOptions)

func WithAddr(addr string) ClientOption {
	return func(o *ClientOptions) {
		o.Addr = addr
	}
}

func WithNEID(neID string) ClientOption {
	return func(o *ClientOptions) {
		o.NEID = neID
	}
}

func WithNEType(neType string) ClientOption {
	return func(o *ClientOptions) {
		o.NEType = neType
	}
}

func WithTLSConfig(tls credentials.TransportCredentials) ClientOption {
	return func(o *ClientOptions) {
		o.TLSConfig = tls
	}
}

// ============================================
// 连接管理
// ============================================

// Connect 连接 NMS
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建 gRPC 连接
	conn, err := c.dial(ctx)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	c.conn = conn
	c.client = pb.NewNMSServiceClient(conn)
	c.connected = true

	// 发送连接请求
	_, err = c.client.Connect(ctx, c.newConnectRequest())
	if err != nil {
		c.conn.Close()
		c.connected = false
		return fmt.Errorf("failed to connect: %w", err)
	}

	if c.onConnected != nil {
		c.onConnected()
	}

	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.client == nil {
		return nil
	}

	_, err := c.client.Disconnect(ctx, &pb.DisconnectRequest{
		NeId: c.opts.NEID,
	})

	if c.conn != nil {
		c.conn.Close()
	}

	c.connected = false

	if c.onDisconnected != nil {
		c.onDisconnected()
	}

	return err
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// ============================================
// 内部方法
// ============================================

func (c *Client) dial(ctx context.Context) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    c.opts.KeepAlive,
			Timeout: 10 * time.Second,
		}),
	}

	if c.opts.TLSConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(c.opts.TLSConfig))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	return grpc.DialContext(ctx, c.addr, opts...)
}

func (c *Client) newConnectRequest() *pb.ConnectRequest {
	return &pb.ConnectRequest{
		NeId:         c.opts.NEID,
		NeType:       c.opts.NEType,
		Ip:           c.opts.IP,
		Port:         c.opts.Port,
		Attrs:        c.opts.Attrs,
		Capabilities: c.opts.Capabilities,
	}
}

// ============================================
// 回调设置
// ============================================

func (c *Client) SetOnConnected(fn func()) {
	c.onConnected = fn
}

func (c *Client) SetOnDisconnected(fn func()) {
	c.onDisconnected = fn
}

func (c *Client) SetOnError(fn func(error)) {
	c.onError = fn
}

// ============================================
// 访问器
// ============================================

func (c *Client) Addr() string {
	return c.addr
}

func (c *Client) NEID() string {
	return c.opts.NEID
}

// ============================================
// 内部接口（客户端使用 proto 定义）
// ============================================

func (c *Client) getClient() pb.NMSServiceClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}