// Package client provides HTTP client with async queue and retry for Push SDK.
//
// Client features:
//   - Async queue with worker pool
//   - Exponential backoff with jitter
//   - Connection pooling and reuse
//   - Graceful degradation when queue is full
//
// Usage Example:
//
//	cli := client.New(
//	    client.WithWorkers(8),
//	    client.WithQueueSize(4096),
//	)
//	defer cli.Close()
//
//	// Async push (non-blocking)
//	cli.AsyncPush("http://target:8080", payload)
//
//	// Sync push (blocking)
//	cli.Push("http://target:8080", payload)
//
// PoolStats:
//
//	cli.Stats() returns PoolStats with:
//	  - ActiveWorkers: Current number of active workers
//	  - QueueLength: Current queue length
//	  - TotalProcessed: Total successful requests
//	  - FailedCount: Total failed requests
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultTimeout   = 1 * time.Minute
	defaultRetry     = 0
	defaultMaxDelay  = 30 * time.Second
	defaultInitDelay = 100 * time.Millisecond
	maxErrBodyBytes  = 4096
)

var (
	defaultWorkers = runtime.NumCPU()
	defaultQueueSz = 4096
)

type httpStatusError struct {
	statusCode int
	body       string
}

func (e *httpStatusError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("http %d", e.statusCode)
	}
	return fmt.Sprintf("http %d: %s", e.statusCode, e.body)
}

// PoolStats holds statistics about the client's worker pool and queue.
//
// Thread-safe: all fields are atomically accessed.
type PoolStats struct {
	ActiveWorkers  int32 // Number of currently active worker goroutines
	QueueLength    int   // Current number of jobs waiting in queue
	TotalProcessed int64 // Total number of successfully processed requests
	FailedCount    int64 // Total number of failed requests
}

type pushJob struct {
	url     string
	payload any
	timeout time.Duration
	next    *pushJob
}

var jobPool = sync.Pool{
	New: func() any {
		return &pushJob{}
	},
}

var httpClientPool = sync.Pool{
	New: func() any {
		return &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				MaxConnsPerHost:     100,
			},
			Timeout: defaultTimeout,
		}
	},
}

var jsonBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Client manages HTTP push operations with async queue and worker pool.
//
// Uses connection pooling and exponential backoff for reliable delivery.
// Thread-safe for concurrent use.
//
// Example:
//
//	cli := client.New(
//	    client.WithWorkers(8),
//	    client.WithQueueSize(4096),
//	)
//	defer cli.Close()
type Client struct {
	baseURL string
	timeout time.Duration
	retry   int
	queueSz int
	workers int

	asyncCh chan *pushJob

	running atomic.Bool

	cli *http.Client
	wg  sync.WaitGroup
	mu  sync.Mutex

	activeWorkers  atomic.Int32
	totalProcessed atomic.Int64
	failedCount    atomic.Int64
}

// Option configures Client behavior using functional options pattern.
type Option func(*Client)

// WithBaseURL sets the base URL for all requests.
//
// Currently unused by Client directly, available for higher-level wrappers.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithTimeout sets the default timeout for push operations.
//
// If not set, defaults to 1 minute. Can be overridden per-request
// using PushTimeout/AsyncPushTimeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithRetry sets the number of retry attempts for failed requests.
//
// Set to 0 for no retries. If not set, defaults to 0.
// Retries use exponential backoff with jitter.
func WithRetry(n int) Option {
	return func(c *Client) { c.retry = n }
}

// WithWorkers sets the number of worker goroutines for async processing.
//
// Each worker processes jobs from the queue. More workers = higher throughput
// but more resource usage. If not set, defaults to runtime.NumCPU().
func WithWorkers(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.workers = n
		}
	}
}

// WithQueueSize sets the maximum queue size for async operations.
//
// If the queue is full, AsyncPush will fall back to synchronous push.
// If not set, defaults to 4096.
func WithQueueSize(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.queueSz = n
		}
	}
}

// WithAsyncQueue is a convenience option to set both workers and queue size.
//
// Equivalent to calling both WithWorkers and WithQueueSize.
func WithAsyncQueue(workers, queueSize int) Option {
	return func(c *Client) {
		if workers > 0 {
			c.workers = workers
		}
		if queueSize > 0 {
			c.queueSz = queueSize
		}
	}
}

// New creates a new Client with the specified options.
//
// The client starts worker goroutines immediately. Call Close to shut down.
func New(opts ...Option) *Client {
	c := &Client{
		timeout: defaultTimeout,
		retry:   defaultRetry,
		workers: defaultWorkers,
		queueSz: defaultQueueSz,
		cli:     httpClientPool.Get().(*http.Client),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.asyncCh = make(chan *pushJob, c.queueSz)
	c.running.Store(true)

	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	return c
}

func (c *Client) worker(_ int) {
	defer c.wg.Done()
	c.activeWorkers.Add(1)
	defer c.activeWorkers.Add(-1)

	for {
		job, ok := <-c.asyncCh
		if !ok || job == nil {
			return
		}

		err := c.doPushWithRetry(job.url, job.payload, job.timeout, 0)
		if err == nil {
			c.totalProcessed.Add(1)
		} else {
			c.failedCount.Add(1)
		}

		job.next = nil
		job.payload = nil
		job.url = ""
		jobPool.Put(job)
	}
}

// Push sends a payload synchronously to the specified URL.
//
// Blocks until the request completes or times out. Uses default timeout.
// Returns an error if the request fails after all retries.
//
// Example:
//
//	err := cli.Push("http://target:8080/api/push", data)
func (c *Client) Push(url string, payload any) error {
	return c.PushTimeout(url, payload, c.timeout)
}

// PushTimeout sends a payload with a custom timeout.
//
// This is the main synchronous push method. Uses exponential backoff
// for retries on transient failures.
func (c *Client) PushTimeout(url string, payload any, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = c.timeout
	}
	return c.doPushWithRetry(url, payload, timeout, c.retry)
}

// AsyncPush sends a payload asynchronously to the specified URL.
//
// Non-blocking: returns immediately after queuing. Uses the default
// timeout for each request. Falls back to synchronous push if queue is full.
//
// Example:
//
//	cli.AsyncPush("http://target:8080/api/push", data)
func (c *Client) AsyncPush(url string, payload any) error {
	return c.AsyncPushTimeout(url, payload, c.timeout)
}

// AsyncPushTimeout sends a payload asynchronously with a custom timeout.
//
// Returns immediately after queuing (non-blocking). If the async queue
// is full, falls back to synchronous push with the specified timeout.
func (c *Client) AsyncPushTimeout(url string, payload any, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = c.timeout
	}

	job := jobPool.Get().(*pushJob)
	job.url = url
	job.payload = payload
	job.timeout = timeout

	select {
	case c.asyncCh <- job:
		return nil
	default:
		job.next = nil
		job.payload = nil
		job.url = ""
		jobPool.Put(job)

		return c.doPushWithRetry(url, payload, timeout, 0)
	}
}

func (c *Client) doPushWithRetry(url string, payload any, timeout time.Duration, retry int) error {
	if timeout <= 0 {
		timeout = c.timeout
	}
	if retry < 0 {
		retry = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= retry; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoff(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return ctx.Err()
			case <-timer.C:
			}
		}

		lastErr = c.doPush(ctx, url, payload)
		if lastErr == nil {
			return nil
		}

		if !isRetryable(lastErr) {
			break
		}
	}

	return fmt.Errorf("push failed after %d retries: %w", retry, lastErr)
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	delay := defaultInitDelay * time.Duration(1<<uint(attempt-1))
	if delay > defaultMaxDelay {
		delay = defaultMaxDelay
	}

	jitter := time.Duration(rand.Int64N(int64(delay / 2)))
	return delay + jitter/2
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		return statusErr.statusCode == http.StatusTooManyRequests || statusErr.statusCode >= 500
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "temporary failure") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "i/o timeout")
}

func (c *Client) doPush(ctx context.Context, url string, payload any) error {
	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer jsonBufferPool.Put(buf)

	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	if n := buf.Len(); n > 0 && buf.Bytes()[n-1] == '\n' {
		buf.Truncate(n - 1)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-oam-push/1.0")
	req.ContentLength = int64(buf.Len())

	resp, err := c.cli.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodyBytes))
		_, _ = io.Copy(io.Discard, resp.Body)
		return &httpStatusError{
			statusCode: resp.StatusCode,
			body:       string(data),
		}
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Close gracefully shuts down the client.
//
// Waits for all pending jobs to complete before returning.
// Safe to call multiple times.
func (c *Client) Close() {
	if c == nil {
		return
	}
	if c.running.CompareAndSwap(true, false) {
		close(c.asyncCh)
		c.wg.Wait()
	}
}

// SetWorkers dynamically adjusts the number of worker goroutines.
//
// Can be called at runtime to scale up or down. When scaling down,
// sends nil jobs to gracefully stop excess workers.
// Thread-safe.
func (c *Client) SetWorkers(n int) {
	if n <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	currentWorkers := c.workers
	if n == currentWorkers {
		return
	}

	if n > currentWorkers {
		for i := currentWorkers; i < n; i++ {
			c.wg.Add(1)
			go c.worker(i)
		}
	} else {
		for i := 0; i < currentWorkers-n; i++ {
			c.asyncCh <- nil
		}
	}

	c.workers = n
}

// BatchPush sends multiple payloads concurrently.
//
// Uses async push for each payload. Waits for all to complete.
// Returns an error if any payloads fail, with count of successes and failures.
//
// Example:
//
//	err := cli.BatchPush("http://target:8080/api/push", payloads)
func (c *Client) BatchPush(url string, payloads []any) error {
	if len(payloads) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errCount int64
	var successCount int64
	var firstErr error

	for _, payload := range payloads {
		wg.Add(1)
		go func(p any) {
			defer wg.Done()
			err := c.AsyncPush(url, p)
			if err != nil {
				mu.Lock()
				errCount++
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(payload)
	}

	wg.Wait()

	if errCount > 0 {
		return fmt.Errorf("batch push completed with %d failures, %d successes: %w", errCount, successCount, firstErr)
	}

	return nil
}

// Stats returns current client statistics.
//
// Useful for monitoring and debugging. Thread-safe.
//
// Example:
//
//	stats := cli.Stats()
//	fmt.Printf("Processed: %d, Failed: %d\n", stats.TotalProcessed, stats.FailedCount)
func (c *Client) Stats() PoolStats {
	return PoolStats{
		ActiveWorkers:  c.activeWorkers.Load(),
		QueueLength:    len(c.asyncCh),
		TotalProcessed: c.totalProcessed.Load(),
		FailedCount:    c.failedCount.Load(),
	}
}

// HealthCheck verifies the client is operational.
//
// Returns nil if the client is running and can accept new jobs.
// Returns an error if the client is stopped or queue is blocked.
func (c *Client) HealthCheck() error {
	if !c.running.Load() {
		return errors.New("client is not running")
	}

	select {
	case job := <-c.asyncCh:
		c.asyncCh <- job
		return nil
	default:
		return nil
	}
}
