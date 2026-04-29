package push

import (
	"fmt"
	"time"
)

// ============ 推送 API ============

// Send 推送通用数据（到默认 OMC 地址）
func (p *Push) Send(record *Record) error {
	return p.SendURL(p.pushURL, record)
}

// SendURL 推送通用数据到自定义 URL
func (p *Push) SendURL(url string, record *Record) error {
	return p.SendURLWithTimeout(url, record, p.timeout)
}

// SendWithTimeout 推送通用数据（指定超时）
func (p *Push) SendWithTimeout(record *Record, timeout time.Duration) error {
	return p.SendURLWithTimeout(p.pushURL, record, timeout)
}

// SendURLWithTimeout 推送通用数据到自定义 URL（指定超时）
func (p *Push) SendURLWithTimeout(url string, record *Record, timeout time.Duration) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	p.hist.HistoryPush(record)
	if timeout <= 0 {
		return p.cli.Push(url, record)
	}
	return p.cli.PushTimeout(url, record, timeout)
}

// SendAsync 异步推送通用数据（到默认 OMC 地址）
func (p *Push) SendAsync(record *Record) error {
	return p.SendAsyncURL(p.pushURL, record)
}

// SendAsyncURL 异步推送通用数据到自定义 URL
func (p *Push) SendAsyncURL(url string, record *Record) error {
	return p.SendAsyncURLTimeout(url, record, p.timeout)
}

// SendAsyncTimeout 异步推送通用数据（指定超时）
func (p *Push) SendAsyncTimeout(record *Record, timeout time.Duration) error {
	return p.SendAsyncURLTimeout(p.pushURL, record, timeout)
}

// SendAsyncURLTimeout 异步推送通用数据到自定义 URL（指定超时）
func (p *Push) SendAsyncURLTimeout(url string, record *Record, timeout time.Duration) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}

	p.hist.HistoryPush(record)
	if timeout <= 0 {
		return p.cli.AsyncPush(url, record)
	}
	return p.cli.AsyncPushTimeout(url, record, timeout)
}
