package codec

import (
	"errors"

	"github.com/tsmask/go-oam/ws/protocol"
	"github.com/tsmask/go-oam/ws/types"
	"google.golang.org/protobuf/proto"
)

// ProtobufCodec Protocol Buffers 编解码器
// 使用 google.golang.org/protobuf 进行序列化
//
// 特点：
//   - 二进制格式，体积最小
//   - 性能最优
//   - 需要预定义 .proto 文件
//   - 跨语言需要生成对应语言代码
//
// 适用场景：高性能服务间通信、微服务
type ProtobufCodec struct{}

// NewProtobufCodec 创建 Protocol Buffers 编解码器
func NewProtobufCodec() Codec {
	return &ProtobufCodec{}
}

// Marshal 将实现了 proto.Message 接口的对象序列化为字节数组
func (c *ProtobufCodec) Marshal(v any) ([]byte, error) {
	if msg, ok := v.(proto.Message); ok {
		return proto.Marshal(msg)
	}
	return nil, errors.New("invalid message type, expected proto.Message")
}

// Unmarshal 将字节数组反序列化到实现了 proto.Message 接口的对象
func (c *ProtobufCodec) Unmarshal(data []byte, v any) error {
	if msg, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, msg)
	}
	return errors.New("invalid message type, expected proto.Message")
}

// Name 实现 Codec 接口
func (c *ProtobufCodec) Name() string {
	return "protobuf"
}

// MessageType 实现 Codec 接口，返回 BinaryMessage
func (c *ProtobufCodec) MessageType() int {
	return BinaryMessage
}

// MarshalRequest 将 types.Request 转换为 protocol.Request 并序列化
func (c *ProtobufCodec) MarshalRequest(req *types.Request) ([]byte, error) {
	pbreq := &protocol.Request{
		Id:     req.ID,
		Action: req.Action,
		Data:   req.Data,
	}
	return proto.Marshal(pbreq)
}

// MarshalResponse 将 types.Response 转换为 protocol.Response 并序列化
func (c *ProtobufCodec) MarshalResponse(resp *types.Response) ([]byte, error) {
	pbresp := &protocol.Response{
		Id:     resp.ID,
		Ts:     resp.Ts,
		Action: resp.Action,
		Code:   resp.Code,
		Msg:    resp.Msg,
		Data:   resp.Data,
	}
	return proto.Marshal(pbresp)
}

// UnmarshalRequest 将字节数组反序列化为 protocol.Request，再转换为 types.Request
func (c *ProtobufCodec) UnmarshalRequest(data []byte) (*types.Request, error) {
	pbreq := &protocol.Request{}
	if err := proto.Unmarshal(data, pbreq); err != nil {
		return nil, err
	}
	return &types.Request{
		ID:     pbreq.GetId(),
		Action: pbreq.GetAction(),
		Data:   pbreq.GetData(),
	}, nil
}

// UnmarshalResponse 将字节数组反序列化为 protocol.Response，再转换为 types.Response
func (c *ProtobufCodec) UnmarshalResponse(data []byte) (*types.Response, error) {
	pbresp := &protocol.Response{}
	if err := proto.Unmarshal(data, pbresp); err != nil {
		return nil, err
	}
	return &types.Response{
		ID:     pbresp.GetId(),
		Ts:     pbresp.GetTs(),
		Action: pbresp.GetAction(),
		Code:   pbresp.GetCode(),
		Msg:    pbresp.GetMsg(),
		Data:   pbresp.GetData(),
	}, nil
}
