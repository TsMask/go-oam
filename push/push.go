package push

import (
	"encoding/json"
	"time"

	"github.com/tsmask/go-oam/push/client"
	"github.com/tsmask/go-oam/push/history"
	"github.com/tsmask/go-oam/push/metrics"
	"github.com/tsmask/go-oam/push/timer"
)

// DefaultPushURI 默认推送路径
const DefaultPushURI = "/api/push/receive"

// Record represents a generic data record for push operations.
//
// Fields:
//   - CoreUID: Core network ID
//   - NeUID: Network element ID
//   - RecordTime: Record timestamp (UTC milliseconds)
//   - RecordType: Record type identifier (e.g., "alarm", "metrics")
//   - RecordData: Flexible data payload (any JSON-serializable type)
//
// Example:
//
//	record := &push.Record{
//	    NeUID:      "ne-001",
//	    RecordType: "alarm",
//	    RecordData: map[string]any{"level": "critical", "message": "CPU overload"},
//	}
type Record struct {
	CoreUID    string            `json:"core_uid,omitempty"`    // Core network ID
	NeUID      string            `json:"ne_uid,omitempty"`      // Network element ID
	RecordTime int64             `json:"record_time,omitempty"` // Record time (UTC milliseconds)
	RecordType string            `json:"record_type,omitempty"` // Record type identifier
	RecordData json.RawMessage   `json:"record_data,omitempty"` // Record data payload
	Params     map[string]string `json:"params,omitempty"`      // Additional parameters
}

// SendParams defines parameters for Send and SendAsync operations.
//
// Timeout interpretation: timeout <= 0 uses client's default timeout.
// URL interpretation: empty URL uses the default push URL.
//
// Example:
//
//	params := &push.SendParams{
//	    URL:     "https://custom-api.com/hook",
//	    Timeout: 5 * time.Second,
//	}
//	p.Send(record, params)
type SendParams struct {
	// URL is the destination endpoint. If empty, uses the default push URL.
	URL string

	// Timeout specifies the request timeout. If <= 0, uses the client's default timeout.
	Timeout time.Duration
}

// Push is the core client for sending data records to push endpoints.
//
// It manages HTTP client connections, retry logic, and provides both synchronous
// and asynchronous send methods. Thread-safe for concurrent use.
//
// Example:
//
//	p := push.New(
//	    push.WithBaseURL("https://api.example.com"),
//	    push.WithTimeout(30 * time.Second),
//	    push.WithRetry(3),
//	)
//	defer p.Close()
//
//	record := &push.Record{
//	    NeUID:      "ne-001",
//	    RecordType: "alarm",
//	    RecordData: map[string]any{"level": "critical"},
//	}
//	p.Send(record, nil)
type Push struct {
	baseURL string
	pushURI string
	pushURL string
	timeout time.Duration
	retry   int
	cli     *client.Client
}

// Option configures Push client behavior using functional options pattern.
//
// See New for usage examples.
type Option func(*Push)

// WithBaseURL sets the base URL for push requests.
//
// The URL should include the protocol and host, e.g., "https://api.example.com".
// If not set, defaults to "http://localhost".
func WithBaseURL(url string) Option {
	return func(p *Push) {
		if url != "" {
			p.baseURL = url
		}
	}
}

// WithPushURI sets the URI path for push requests.
//
// The URI should start with "/" and include the endpoint path.
// If not set, defaults to "/api/push/receive".
//
// Example:
//
//	WithPushURI("/api/v1/alarms")
func WithPushURI(uri string) Option {
	return func(p *Push) {
		if uri != "" {
			p.pushURI = uri
		}
	}
}

// WithTimeout sets the request timeout for push operations.
//
// If not set, defaults to 1 minute. The timeout applies to each individual
// HTTP request, not the total operation time.
func WithTimeout(d time.Duration) Option {
	return func(p *Push) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithRetry sets the number of retry attempts for failed requests.
//
// Set to 0 for no retries (fail-fast). If not set, defaults to 0.
// Retries are performed for transient network errors and 5xx responses.
func WithRetry(n int) Option {
	return func(p *Push) {
		if n >= 0 {
			p.retry = n
		}
	}
}

// New creates a new Push client with optional configuration.
//
// The client must be closed after use to release resources.
//
// Example:
//
//	// Basic usage with defaults
//	p := push.New()
//
//	// Custom configuration
//	p := push.New(
//	    push.WithBaseURL("https://api.example.com"),
//	    push.WithPushURI("/api/v1/push"),
//	    push.WithTimeout(30 * time.Second),
//	    push.WithRetry(3),
//	)
//	defer p.Close()
func New(opts ...Option) *Push {
	p := &Push{
		timeout: 1 * time.Minute,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.baseURL == "" {
		p.baseURL = "http://localhost"
	}
	if p.pushURI == "" {
		p.pushURI = DefaultPushURI
	}
	p.pushURL = p.baseURL + p.pushURI

	p.cli = client.New(
		client.WithTimeout(p.timeout),
		client.WithRetry(p.retry),
	)

	return p
}

// Close releases all resources held by the Push client.
//
// It should be called when the client is no longer needed.
// Safe to call multiple times.
func (p *Push) Close() {
	if p.cli != nil {
		p.cli.Close()
	}
}

// Send synchronously sends a record to the push endpoint.
//
// Blocks until the request completes or times out. If record.RecordTime is zero,
// it will be set to the current UTC time.
//
// Parameters:
//   - record: The data record to send
//   - params: Send parameters (nil for defaults: URL uses baseURL+pushURI, Timeout uses client default)
//
// Returns an error if the request fails after all retries.
//
// Example:
//
//	record := &push.Record{
//	    NeUID:      "ne-001",
//	    RecordType: "metrics",
//	    RecordData: map[string]any{"cpu": 85.5},
//	}
//	if err := p.Send(record, nil); err != nil {
//	    log.Printf("send failed: %v", err)
//	}
//
//	// With custom URL and timeout
//	err := p.Send(record, &push.SendParams{
//	    URL:     "https://custom-api.com/hook",
//	    Timeout: 5 * time.Second,
//	})
func (p *Push) Send(record *Record, params *SendParams) error {
	url := p.pushURL
	timeout := p.timeout

	if params != nil {
		if params.URL != "" {
			url = params.URL
		}
		if params.Timeout > 0 {
			timeout = params.Timeout
		}
	}

	if record.RecordTime == 0 {
		record.RecordTime = time.Now().UnixMilli()
	}
	return p.cli.PushTimeout(url, record, timeout)
}

// SendAsync sends a record asynchronously to the push endpoint.
//
// Non-blocking: returns immediately after queuing the request.
// Uses an internal goroutine pool for efficient concurrent execution.
// Returns an error if the record cannot be queued.
//
// Parameters:
//   - record: The data record to send
//   - params: Send parameters (nil for defaults: URL uses baseURL+pushURI, Timeout uses client default)
//
// Example:
//
//	// Fire and forget
//	if err := p.SendAsync(record, nil); err != nil {
//	    log.Printf("queue failed: %v", err)
//	}
//
//	// With custom URL and timeout
//	err := p.SendAsync(record, &push.SendParams{
//	    URL:     "https://custom-api.com/async-hook",
//	    Timeout: 10 * time.Second,
//	})
func (p *Push) SendAsync(record *Record, params *SendParams) error {
	url := p.pushURL
	timeout := p.timeout

	if params != nil {
		if params.URL != "" {
			url = params.URL
		}
		if params.Timeout > 0 {
			timeout = params.Timeout
		}
	}

	if record.RecordTime == 0 {
		record.RecordTime = time.Now().UnixMilli()
	}
	return p.cli.AsyncPushTimeout(url, record, timeout)
}

// NewMetrics creates a new non-sharded metrics collector.
//
// Non-sharded version is simpler but may have lock contention under
// high concurrency. For better performance with concurrent writes,
// use NewShardedMetrics.
//
// Example:
//
//	m := push.NewMetrics()
//	m.Incr("requests_total")
func NewMetrics() *metrics.Metrics {
	return metrics.New()
}

// NewShardedMetrics creates a new sharded metrics collector.
//
// Uses multiple shards to reduce lock contention. Each shard has its
// own lock, allowing better concurrent write performance.
//
// Example:
//
//	m := push.NewShardedMetrics()
//	m.Incr("requests_total")
func NewShardedMetrics() *metrics.ShardedMetrics {
	return metrics.NewSharded()
}

// NewTimer creates a a new timer for measuring operation durations.
//
// Useful for performance monitoring and profiling.
//
// Example:
//
//	t := push.NewTimer()
//	defer t.Stop("operation_name")
func NewTimer() *timer.Timer {
	return timer.New()
}

// NewHistory creates a new history buffer with the specified maximum size.
//
// The history buffer stores records for debugging, auditing, or replay.
// Set maxSize to control memory usage. Set to 0 or negative to disable.
//
// Example:
//
//	h := push.NewHistory[push.Record](1000)
//	h.Record(record)
func NewHistory[V any](maxSize int) *history.History[V] {
	return history.New[V](maxSize)
}

// NewShardedHistory creates a new sharded history buffer.
//
// Uses multiple shards to reduce lock contention for high-throughput
// scenarios. Each shard has its own ring buffer.
//
// Example:
//
//	sh := push.NewShardedHistory[push.Record](100)
func NewShardedHistory[V any](maxSize int) *history.ShardedHistory[V] {
	return history.NewSharded[V](maxSize)
}

// NewClient creates a new HTTP client for custom use cases.
//
// Provides access to the underlying HTTP client with configurable
// timeout and retry behavior.
//
// Example:
//
//	cli := push.NewClient(
//	    client.WithTimeout(30 * time.Second),
//	    client.WithRetry(3),
//	)
func NewClient(opts ...client.Option) *client.Client {
	return client.New(opts...)
}
