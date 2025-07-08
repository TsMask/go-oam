package fetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tsmask/go-oam/src/framework/config"
)

// userAgent 自定义 User-Agent
var userAgent = fmt.Sprintf("%s/%s", strings.ToLower(fmt.Sprint(config.Get("ne.type"))), config.Get("ne.version"))

// Get 发送 GET 请求
// timeout 超时时间（毫秒）
func Get(url string, headers map[string]string, timeout int) ([]byte, error) {
	if timeout < 100 || timeout > 180_000 {
		timeout = 100
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond, // 超时时间
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Post 发送 POST 请求
func Post(url string, data url.Values, headers map[string]string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// PostJSON 发送 POST 请求，并将请求体序列化为 JSON 格式
func PostJSON(url string, data any, headers map[string]string) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // 超时时间
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
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

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return responseBody, nil
}

// PutJSON 发送 PUT 请求，并将请求体序列化为 JSON 格式
func PutJSON(url string, data any, headers map[string]string) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // 超时时间
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Delete 发送 DELETE 请求
func Delete(url string, headers map[string]string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
