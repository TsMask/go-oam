package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// Options 请求选项
type Options struct {
	Ctx       context.Context   // 上下文，用于取消/超时控制
	Headers   map[string]string // 自定义请求头
	Query     map[string]string // 查询参数
	Form      map[string]string // 表单数据
	Files     []FileUpload      // 文件上传列表
	JSON      any               // JSON Body
	Debug     bool              // 是否打印调试日志
	LocalAddr string            // 发起请求的源 IP 地址（多网卡场景），空值使用系统默认
}

// FileUpload 文件上传项
type FileUpload struct {
	Field  string    // 表单字段名
	Reader io.Reader // 文件数据流，优先使用，适合大文件（与 Data/Path 三选一）
	Path   string    // 文件绝对路径
	Data   *[]byte   // 文件数据（优先级高于 Path，低于 Reader）
	Name   string    // 文件名（使用 Reader/Data 时必填，缺省取 Field）
}

// 全局复用 Client（无源 IP 绑定）
var baseClient = newClient("")

// 带源 IP 的 client 缓存，按 IP 复用连接池
var localClients sync.Map

// getClient 获取 HTTP 客户端（带源 IP 缓存复用）
func getClient(localAddr string) *resty.Client {
	if localAddr == "" {
		return baseClient
	}
	if v, ok := localClients.Load(localAddr); ok {
		return v.(*resty.Client)
	}
	client := newClient(localAddr)
	actual, _ := localClients.LoadOrStore(localAddr, client)
	return actual.(*resty.Client)
}

// newClient 创建 resty 客户端（localAddr 为空时使用系统默认路由）
func newClient(localAddr string) *resty.Client {
	transport := &http.Transport{
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:  10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	// 绑定源 IP 发起连接
	if localAddr != "" {
		if localIP := net.ParseIP(localAddr); localIP != nil {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				d := &net.Dialer{
					LocalAddr: &net.TCPAddr{IP: localIP},
				}
				return d.DialContext(ctx, network, addr)
			}
		}
	}

	return resty.New().
		SetTransport(transport).
		SetRetryCount(2).
		SetRetryWaitTime(300 * time.Millisecond).
		SetRetryMaxWaitTime(5 * time.Second).
		SetTimeout(3 * time.Minute).
		SetCloseConnection(false)
}

// build 构建单次请求
func build(opts Options) *resty.Request {
	client := getClient(opts.LocalAddr)
	req := client.R()
	req.SetDebug(opts.Debug)

	if opts.Ctx != nil {
		req.SetContext(opts.Ctx)
	}
	if len(opts.Headers) > 0 {
		req.SetHeaders(opts.Headers)
	}
	if len(opts.Query) > 0 {
		req.SetQueryParams(opts.Query)
	}
	if opts.JSON != nil {
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(opts.JSON)
	}
	if len(opts.Form) > 0 {
		req.SetFormData(opts.Form)
	}

	// 文件上传：Reader > Data > Path
	for i := range opts.Files {
		f := &opts.Files[i]
		name := f.Name
		if name == "" {
			name = f.Field
		}
		if f.Reader != nil {
			req.SetFileReader(f.Field, name, f.Reader)
		} else if f.Data != nil && len(*f.Data) > 0 {
			req.SetFileReader(f.Field, name, bytes.NewReader(*f.Data))
		} else if f.Path != "" {
			req.SetFile(f.Field, f.Path)
		}
	}

	return req
}

// do 统一执行请求并处理响应
func do(req *resty.Request, method, url string) ([]byte, error) {
	resp, err := req.Execute(method, url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s %s: %w", method, url, err)
	}
	if resp.IsError() {
		return resp.Body(), fmt.Errorf("fetch %s %s: HTTP %d", method, url, resp.StatusCode())
	}
	return resp.Body(), nil
}

// Get 发送 GET 请求
func Get(url string, opts Options) ([]byte, error) {
	return do(build(opts), http.MethodGet, url)
}

// Post 发送 POST 请求
func Post(url string, opts Options) ([]byte, error) {
	return do(build(opts), http.MethodPost, url)
}

// Put 发送 PUT 请求
func Put(url string, opts Options) ([]byte, error) {
	return do(build(opts), http.MethodPut, url)
}

// Delete 发送 DELETE 请求
func Delete(url string, opts Options) ([]byte, error) {
	return do(build(opts), http.MethodDelete, url)
}
