package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/utils/parse"
)

// userAgent 自定义 User-Agent
var userAgent = fmt.Sprintf("%s/%s", strings.ToLower(fmt.Sprint(config.Get("ne.type"))), config.Get("ne.version"))

var defaultTransport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 100,
	IdleConnTimeout:     90 * time.Second,
	ForceAttemptHTTP2:   true, // 连接池与 HTTP/2
}

var defaultClient = &http.Client{Transport: defaultTransport}

func maxResponseBytes() int64 {
	mb := parse.Number(config.Get("fetch.maxResponseMB"))
	if mb <= 0 {
		mb = 4
	}
	return int64(mb) * 1024 * 1024
}

// Get 发送 GET 请求
func Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes()))
	if err != nil {
		return nil, err
	}

	return body, nil
}

// PostForm 发送 POST 请求, 并将请求体序列化为表单格式
func PostForm(ctx context.Context, url string, data url.Values, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes()))
	if err != nil {
		return nil, err
	}

	return body, nil
}

// PostJSON 发送 POST 请求，并将请求体序列化为 JSON 格式
func PostJSON(ctx context.Context, url string, data any, headers map[string]string) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var lastErr error
	backoff := 200 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		resp, err := defaultClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			defer resp.Body.Close()
			if resp.StatusCode >= http.StatusInternalServerError {
				lastErr = fmt.Errorf("request returned status: %s", resp.Status)
			} else if resp.StatusCode >= http.StatusMultipleChoices {
				return nil, fmt.Errorf("request returned status: %s", resp.Status)
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					lastErr = err
				} else {
					return body, nil
				}
			}
		}
		if attempt < 2 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return nil, lastErr
}

// UploadFile 上传文件函数，接收 URL 地址、表单参数和文件对象，返回响应内容或错误信息
func PostUploadFile(url string, params map[string]string, file *os.File) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file content: %v", err)
	}

	for key, value := range params {
		err = writer.WriteField(key, value)
		if err != nil {
			return nil, fmt.Errorf("failed to write form field: %v", err)
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return responseBody, nil
}

// PutJSON 发送 PUT 请求，并将请求体序列化为 JSON 格式
func PutJSON(ctx context.Context, url string, data any, headers map[string]string) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes()))
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Delete 发送 DELETE 请求
func Delete(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes()))
	if err != nil {
		return nil, err
	}

	return body, nil
}
