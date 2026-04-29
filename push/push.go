package push

import (
	"strings"
	"time"

	"github.com/tsmask/go-oam/push/internal"
)

// DefaultPushURI 默认推送路径
const DefaultPushURI = "/api/push/receive"

// Record 通用数据
type Record = internal.Record

// Push 推送功能集
// 统一入口，所有数据类型通过 Type 字段区分
type Push struct {
	baseURL     string        // OMC 网管地址
	pushURI     string        // 推送路径（默认 /push/receive）
	pushURL     string        // 预计算：baseURL + pushURI，避免每次拼接
	timeout     time.Duration // 默认超时时间（默认值1分钟）
	histMaxSize int           // 历史记录上限（默认值1024）
	retry       int           // 重试次数（默认值0）
	cli         *internal.Client
	hist        *internal.History
}

// Option 配置选项
type Option func(*Push)

// WithBaseURL 设置 OMC 网管地址
// 同时预计算默认推送 URL，减少每次推送的开销
func WithBaseURL(url string) Option {
	return func(p *Push) {
		p.baseURL = strings.TrimSuffix(url, "/")
		if p.pushURI == "" {
			p.pushURI = DefaultPushURI
		}
		p.pushURL = p.baseURL + p.pushURI
	}
}

// WithPushURI 设置推送路径（默认 /push/receive）
func WithPushURI(uri string) Option {
	return func(p *Push) {
		if strings.HasPrefix(uri, "/") {
			p.pushURI = uri
		}
	}
}

// WithTimeout 设置默认超时时间（默认值1分钟）
func WithTimeout(d time.Duration) Option {
	return func(p *Push) { p.timeout = d }
}

// WithRetry 设置重试次数（默认 0，即不重试）
func WithRetry(n int) Option {
	return func(p *Push) {
		if n >= 0 {
			p.retry = n
		}
	}
}

// WithHistoryMaxSize 设置历史记录上限（默认 1024）
func WithHistoryMaxSize(maxSize int) Option {
	return func(p *Push) {
		if maxSize > 0 {
			p.histMaxSize = maxSize
		}
	}
}

// New 创建 Push 实例
func New(opts ...Option) *Push {
	p := &Push{}

	for _, opt := range opts {
		opt(p)
	}

	// 确保 baseURL 有默认值
	if p.baseURL == "" {
		p.baseURL = "http://localhost"
	}
	// 确保使用默认推送路径
	if p.pushURI == "" {
		p.pushURI = DefaultPushURI
	}
	// 计算完整推送 URL
	p.pushURL = p.baseURL + p.pushURI

	// 初始化 HTTP 客户端
	if p.cli == nil {
		opts := []internal.Option{internal.WithBaseURL(p.baseURL)}
		if p.timeout > 0 {
			opts = append(opts, internal.WithTimeout(p.timeout))
		}
		if p.retry > 0 {
			opts = append(opts, internal.WithRetry(p.retry))
		}
		p.cli = internal.NewClient(opts...)
	}

	// 初始化历史记录服务
	p.hist = internal.NewHistory(p.histMaxSize)

	return p
}

// Close 关闭客户端，释放资源
func (p *Push) Close() {
	if p.cli != nil {
		p.cli.Close()
	}
}

// NewMetrics 创建独立的指标管理器
// 可以创建多个实例分别统计不同指标
func NewMetrics() *internal.Metrics {
	return internal.NewMetrics()
}

// NewTimer 创建独立的周期定时器
// 可以创建多个实例分别管理不同的周期任务
func NewTimer() *internal.Timer {
	return internal.NewTimer()
}
