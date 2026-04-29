package internal

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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultTimeout   = 1 * time.Minute
	defaultWorkers   = 4
	defaultQueueSz   = 1024
	defaultRetry     = 0
	defaultMaxDelay  = 30 * time.Second
	defaultInitDelay = 100 * time.Millisecond
	maxErrBodyBytes  = 4096
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
}

type Option func(*Client)

func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

func WithRetry(n int) Option {
	return func(c *Client) { c.retry = n }
}

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

func NewClient(opts ...Option) *Client {
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
	for {
		job, ok := <-c.asyncCh
		if !ok {
			return
		}

		c.doPushWithRetry(job.url, job.payload, job.timeout, 0)

		job.next = nil
		job.payload = nil
		job.url = ""
		jobPool.Put(job)
	}
}

func (c *Client) Push(url string, payload any) error {
	return c.PushTimeout(url, payload, c.timeout)
}

func (c *Client) PushTimeout(url string, payload any, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = c.timeout
	}
	return c.doPushWithRetry(url, payload, timeout, c.retry)
}

func (c *Client) AsyncPush(url string, payload any) error {
	return c.AsyncPushTimeout(url, payload, c.timeout)
}

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
	// Encoder 默认会追加换行
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

func (c *Client) Close() {
	if c == nil {
		return
	}
	if c.running.CompareAndSwap(true, false) {
		close(c.asyncCh)
		c.wg.Wait()
	}
}
