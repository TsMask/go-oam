package processor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/utils/file"
)

// FileUpload 文件上传
func FileUpload(cfg *config.Config, messageType int, data []byte) (map[string]string, error) {
	var body struct {
		FileName string `json:"fileName"` // 文件名
		File     []byte `json:"file"`     // 文件内容 文本消息是base64字符
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("message data format error")
	}
	if body.FileName == "" || body.File == nil {
		return nil, fmt.Errorf("fileName and file must be set")
	}

	// 文本消息需要解码
	var bin []byte = body.File
	if messageType == websocket.TextMessage {
		v, err := base64.StdEncoding.DecodeString(string(body.File))
		if err != nil {
			return nil, err
		}
		bin = v
	}

	uploadFilePath, err := file.TransferUploadBytes(cfg, body.FileName, bin, []string{})
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"filePath":         uploadFilePath,
		"newFileName":      filepath.Base(uploadFilePath),
		"originalFileName": body.FileName,
	}, nil
}
