package codec

import (
	"encoding/json"

	"github.com/tsmask/go-oam/ws/types"
)

// jsonCodec JSON 编解码器
// 使用标准库 encoding/json 进行序列化
//
// 特点：
//   - 人类可读，便于调试
//   - 跨语言兼容性好
//   - 性能一般，体积较大
//
// 适用场景：开发调试、跨语言通信
type jsonCodec struct{}

// Marshal 实现 Codec 接口
func (c *jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 实现 Codec 接口
func (c *jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// Name 实现 Codec 接口
func (c *jsonCodec) Name() string { return "json" }

// MessageType 实现 Codec 接口，返回 TextMessage
func (c *jsonCodec) MessageType() int { return TextMessage }

// MarshalRequest 实现 Codec 接口
func (c *jsonCodec) MarshalRequest(req *types.Request) ([]byte, error) {
	return json.Marshal(req)
}

// MarshalResponse 实现 Codec 接口
func (c *jsonCodec) MarshalResponse(resp *types.Response) ([]byte, error) {
	return json.Marshal(resp)
}

// UnmarshalRequest 实现 Codec 接口
func (c *jsonCodec) UnmarshalRequest(data []byte) (*types.Request, error) {
	req := &types.Request{}
	if err := json.Unmarshal(data, req); err != nil {
		return nil, err
	}
	return req, nil
}

// UnmarshalResponse 实现 Codec 接口
func (c *jsonCodec) UnmarshalResponse(data []byte) (*types.Response, error) {
	resp := &types.Response{}
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
