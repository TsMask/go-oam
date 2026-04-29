package codec

import (
	"encoding/json"

	"github.com/tsmask/go-oam/ws/types"
)

// JSONCodec JSON 编解码器
// 使用标准库 encoding/json 进行序列化
//
// 特点：
//   - 人类可读，便于调试
//   - 跨语言兼容性好
//   - 性能一般，体积较大
//
// 适用场景：开发调试、跨语言通信
type JSONCodec struct{}

// NewJSONCodec 创建 JSON 编解码器
func NewJSONCodec() Codec {
	return &JSONCodec{}
}

// Marshal 实现 Codec 接口
func (c *JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 实现 Codec 接口
func (c *JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// Name 实现 Codec 接口
func (c *JSONCodec) Name() string {
	return "json"
}

// MessageType 实现 Codec 接口，返回 TextMessage
func (c *JSONCodec) MessageType() int {
	return TextMessage
}

// MarshalRequest 实现 Codec 接口
func (c *JSONCodec) MarshalRequest(req *types.Request) ([]byte, error) {
	return json.Marshal(req)
}

// MarshalResponse 实现 Codec 接口
func (c *JSONCodec) MarshalResponse(resp *types.Response) ([]byte, error) {
	return json.Marshal(resp)
}

// UnmarshalRequest 实现 Codec 接口
func (c *JSONCodec) UnmarshalRequest(data []byte) (*types.Request, error) {
	req := &types.Request{}
	return req, json.Unmarshal(data, req)
}

// UnmarshalResponse 实现 Codec 接口
func (c *JSONCodec) UnmarshalResponse(data []byte) (*types.Response, error) {
	resp := &types.Response{}
	return resp, json.Unmarshal(data, resp)
}