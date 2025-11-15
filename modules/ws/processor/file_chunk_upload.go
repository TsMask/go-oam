package processor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/utils/file"
)

// FileChunkUpload 文件分片上传
func FileChunkUpload(messageType int, data []byte) (map[string]string, error) {
	var body struct {
		FileName   string `json:"fileName"`   // 文件名
		Identifier string `json:"identifier"` // 分片标识
		Index      string `json:"index"`      // 分片序号
		File       []byte `json:"file"`       // 文件内容 文本消息是base64字符
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("message data format error")
	}
	if body.FileName == "" || body.File == nil {
		return nil, fmt.Errorf("fileName and file must be set")
	}
	if body.Index == "" || body.Identifier == "" {
		return nil, fmt.Errorf("index and identifier must be set")
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

	chunkFilePath, err := file.TransferChunkUploadBytes(body.FileName, body.Index, body.Identifier, bin)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"filePath":         chunkFilePath,
		"newFileName":      filepath.Base(chunkFilePath),
		"originalFileName": body.FileName,
	}, nil
}
