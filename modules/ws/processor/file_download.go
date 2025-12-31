package processor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/utils/file"
)

// FileDownload 文件下载
func FileDownload(cfg *config.Config, messageType int, data []byte) (map[string]any, error) {
	var body struct {
		FilePath string `json:"filePath"`
		Range    string `json:"range"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("message data format error")
	}
	if body.FilePath == "" {
		return nil, fmt.Errorf("filePath must be set")
	}
	res, err := file.ReadUploadFileStream(cfg, body.FilePath, body.Range)
	if err != nil {
		return nil, err
	}

	// 处理数据
	var raw []byte
	if b, ok := res["data"].([]byte); ok {
		raw = b
	}
	if messageType == websocket.TextMessage && len(raw) > 0 {
		res["data"] = base64.StdEncoding.EncodeToString(raw)
	}
	return res, nil
}
