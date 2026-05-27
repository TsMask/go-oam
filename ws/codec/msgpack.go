package codec

import (
	"github.com/tsmask/go-oam/ws/types"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// msgpackCodec MessagePack 编解码器
// 使用 vmihailenco/msgpack 进行序列化
//
// 特点：
//   - 二进制格式，体积小
//   - 性能优于 JSON
//   - 跨语言兼容性好
//
// 适用场景：高性能通信、游戏后端
type msgpackCodec struct{}

// Marshal 实现 Codec 接口
func (c *msgpackCodec) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Unmarshal 实现 Codec 接口
func (c *msgpackCodec) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

// Name 实现 Codec 接口
func (c *msgpackCodec) Name() string { return "msgpack" }

// MessageType 实现 Codec 接口，返回 BinaryMessage
func (c *msgpackCodec) MessageType() int { return BinaryMessage }

// MarshalRequest 实现 Codec 接口
func (c *msgpackCodec) MarshalRequest(req *types.Request) ([]byte, error) {
	return msgpack.Marshal(req)
}

// MarshalResponse 实现 Codec 接口
func (c *msgpackCodec) MarshalResponse(resp *types.Response) ([]byte, error) {
	return msgpack.Marshal(resp)
}

// UnmarshalRequest 实现 Codec 接口
func (c *msgpackCodec) UnmarshalRequest(data []byte) (*types.Request, error) {
	req := &types.Request{}
	if err := msgpack.Unmarshal(data, req); err != nil {
		return nil, err
	}
	return req, nil
}

// UnmarshalResponse 实现 Codec 接口
func (c *msgpackCodec) UnmarshalResponse(data []byte) (*types.Response, error) {
	resp := &types.Response{}
	if err := msgpack.Unmarshal(data, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
