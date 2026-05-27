package codec

import (
	"github.com/tsmask/go-oam/ws/types"
)

// 消息类型常量
const (
	TextMessage   = 1 // 文本消息（WebSocket TextMessage）
	BinaryMessage = 2 // 二进制消息（WebSocket BinaryMessage）
)

// 默认 JSON 编解码器（无状态，全局复用）
var defaultJSON Codec = &jsonCodec{}

// JSON 返回全局 JSON 编解码器（无状态，可直接复用）
func JSON() Codec { return defaultJSON }

// Codec 消息编解码器接口
type Codec interface {
	// Marshal 将对象序列化为字节数组
	Marshal(v any) ([]byte, error)

	// Unmarshal 将字节数组反序列化为对象
	Unmarshal(data []byte, v any) error

	// Name 返回编解码器名称
	Name() string

	// MessageType 返回 WebSocket 消息类型（TextMessage 或 BinaryMessage）
	MessageType() int

	// MarshalRequest 将 Request 序列化为字节数组
	MarshalRequest(req *types.Request) ([]byte, error)

	// MarshalResponse 将 Response 序列化为字节数组
	MarshalResponse(resp *types.Response) ([]byte, error)

	// UnmarshalRequest 将字节数组反序列化为 Request
	UnmarshalRequest(data []byte) (*types.Request, error)

	// UnmarshalResponse 将字节数组反序列化为 Response
	UnmarshalResponse(data []byte) (*types.Response, error)
}

// NewCodec 根据名称创建编解码器
// 支持: "json", "msgpack", "protobuf"，默认 "json"
func NewCodec(name string) Codec {
	switch name {
	case "msgpack":
		return &msgpackCodec{}
	case "protobuf":
		return &protobufCodec{}
	default:
		return defaultJSON
	}
}
