package codec

import (
	msgpack "github.com/vmihailenco/msgpack/v5"
	"github.com/tsmask/go-oam/ws/types"
)

// MsgPackCodec MessagePack 编解码器
// 使用 vmihailenco/msgpack 进行序列化
//
// 特点：
//   - 二进制格式，体积小
//   - 性能优于 JSON
//   - 跨语言兼容性好
//
// 适用场景：高性能通信、游戏后端
type MsgPackCodec struct{}

// NewMsgPackCodec 创建 MessagePack 编解码器
func NewMsgPackCodec() Codec {
	return &MsgPackCodec{}
}

// Marshal 实现 Codec 接口
func (c *MsgPackCodec) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Unmarshal 实现 Codec 接口
func (c *MsgPackCodec) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

// Name 实现 Codec 接口
func (c *MsgPackCodec) Name() string {
	return "msgpack"
}

// MessageType 实现 Codec 接口，返回 BinaryMessage
func (c *MsgPackCodec) MessageType() int {
	return BinaryMessage
}

// MarshalRequest 实现 Codec 接口
func (c *MsgPackCodec) MarshalRequest(req *types.Request) ([]byte, error) {
	return msgpack.Marshal(req)
}

// MarshalResponse 实现 Codec 接口
func (c *MsgPackCodec) MarshalResponse(resp *types.Response) ([]byte, error) {
	return msgpack.Marshal(resp)
}

// UnmarshalRequest 实现 Codec 接口
func (c *MsgPackCodec) UnmarshalRequest(data []byte) (*types.Request, error) {
	req := &types.Request{}
	if err := msgpack.Unmarshal(data, req); err != nil {
		return nil, err
	}
	return req, nil
}

// UnmarshalResponse 实现 Codec 接口
func (c *MsgPackCodec) UnmarshalResponse(data []byte) (*types.Response, error) {
	resp := &types.Response{}
	if err := msgpack.Unmarshal(data, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CompactMsgPackCodec 紧凑型 MessagePack 编解码器
// 与 MsgPackCodec 使用相同的 msgpack 库
// 仅名称不同，用于区分不同的序列化配置
//
// 适用场景：需要与标准 msgpack 格式区分的场景
type CompactMsgPackCodec struct{}

// NewCompactMsgPackCodec 创建紧凑型 MessagePack 编解码器
func NewCompactMsgPackCodec() Codec {
	return &CompactMsgPackCodec{}
}

// Marshal 实现 Codec 接口
func (c *CompactMsgPackCodec) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Unmarshal 实现 Codec 接口
func (c *CompactMsgPackCodec) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

// Name 实现 Codec 接口
func (c *CompactMsgPackCodec) Name() string {
	return "msgpack-compact"
}

// MessageType 实现 Codec 接口，返回 BinaryMessage
func (c *CompactMsgPackCodec) MessageType() int {
	return BinaryMessage
}

// MarshalRequest 实现 Codec 接口
func (c *CompactMsgPackCodec) MarshalRequest(req *types.Request) ([]byte, error) {
	return msgpack.Marshal(req)
}

// MarshalResponse 实现 Codec 接口
func (c *CompactMsgPackCodec) MarshalResponse(resp *types.Response) ([]byte, error) {
	return msgpack.Marshal(resp)
}

// UnmarshalRequest 实现 Codec 接口
func (c *CompactMsgPackCodec) UnmarshalRequest(data []byte) (*types.Request, error) {
	req := &types.Request{}
	if err := msgpack.Unmarshal(data, req); err != nil {
		return nil, err
	}
	return req, nil
}

// UnmarshalResponse 实现 Codec 接口
func (c *CompactMsgPackCodec) UnmarshalResponse(data []byte) (*types.Response, error) {
	resp := &types.Response{}
	if err := msgpack.Unmarshal(data, resp); err != nil {
		return nil, err
	}
	return resp, nil
}