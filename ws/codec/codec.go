package codec

import (
	"github.com/tsmask/go-oam/ws/types"
)

// 消息类型常量
const (
	TextMessage   = 1 // 文本消息（WebSocket TextMessage）
	BinaryMessage = 2 // 二进制消息（WebSocket BinaryMessage）
)

// Codec 消息编解码器接口
// 用于 WebSocket 消息的序列化和反序列化
//
// 实现者：
//   - JSONCodec: JSON 编解码器，适合调试和跨语言场景
//   - MsgPackCodec: MessagePack 编解码器，体积小性能好
//   - ProtobufCodec: Protocol Buffers 编解码器，性能最优
type Codec interface {
	// Marshal 将对象序列化为字节数组
	Marshal(v any) ([]byte, error)

	// Unmarshal 将字节数组反序列化为对象
	Unmarshal(data []byte, v any) error

	// Name 返回编解码器名称
	Name() string

	// MessageType 返回 WebSocket 消息类型
	// 返回 TextMessage 或 BinaryMessage
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
