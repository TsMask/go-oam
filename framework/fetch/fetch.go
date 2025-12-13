package fetch

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// Options 请求选项
type Options struct {
	Ctx     context.Context   // 上下文，用于取消请求
	Timeout int               // 单请求超时 (ms)
	Headers map[string]string // 自定义请求头
	Query   map[string]string // 查询参数
	Form    map[string]string // 表单
	Files   []FileUpload      // 文件上传
	JSON    any               // JSON Body
	Debug   bool              // 是否打印 Debug Log
}

// FileUpload 文件上传
type FileUpload struct {
	Field string // 表单字段名
	Path  string // 文件绝对路径
}

// ---------------------
// 全局复用 Client
// ---------------------
var baseClient = resty.New().
	SetRetryCount(2).
	SetRetryWaitTime(300 * time.Millisecond).
	SetRetryMaxWaitTime(2 * time.Second).
	SetTimeout(1 * time.Minute)

// -------------------------
// 构建 Request
// -------------------------
func build(opts Options) (*resty.Client, *resty.Request) {
	client := baseClient

	// 单次请求 Timeout —— 使用 Clone
	if opts.Timeout > 0 {
		client = baseClient.Clone()
		client.SetTimeout(time.Duration(opts.Timeout) * time.Millisecond)
	}

	req := client.R()
	req.SetDebug(opts.Debug)

	if opts.Ctx != nil {
		req.SetContext(opts.Ctx)
	}

	if opts.Headers != nil {
		req.SetHeaders(opts.Headers)
	}

	if opts.Query != nil {
		req.SetQueryParams(opts.Query)
	}

	if opts.JSON != nil {
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(opts.JSON)
	}

	if opts.Form != nil {
		req.SetFormData(opts.Form)
	}

	for _, f := range opts.Files {
		req.SetFile(f.Field, f.Path)
	}

	return client, req
}

// ----------------------
// 统一执行 + 错误处理
// ----------------------
func do(req *resty.Request, method, url string) ([]byte, error) {
	var (
		resp *resty.Response
		err  error
	)

	switch method {
	case resty.MethodGet:
		resp, err = req.Get(url)
	case resty.MethodPost:
		resp, err = req.Post(url)
	case resty.MethodPut:
		resp, err = req.Put(url)
	case resty.MethodDelete:
		resp, err = req.Delete(url)
	}

	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}

	if resp.IsError() {
		return resp.Body(), fmt.Errorf("http error: %s", resp.Status())
	}

	return resp.Body(), nil
}

// ----------------------
// REST 方法封装
// ----------------------

func Get(url string, opts Options) ([]byte, error) {
	_, req := build(opts)
	return do(req, resty.MethodGet, url)
}

func Post(url string, opts Options) ([]byte, error) {
	_, req := build(opts)
	return do(req, resty.MethodPost, url)
}

func Put(url string, opts Options) ([]byte, error) {
	_, req := build(opts)
	return do(req, resty.MethodPut, url)
}

func Delete(url string, opts Options) ([]byte, error) {
	_, req := build(opts)
	return do(req, resty.MethodDelete, url)
}
