package ws

import (
	"github.com/tsmask/go-oam/ws/codec"
)

// NewJSONCodec 创建 JSON 编解码器
// 使用 JSON 格式进行消息序列化和反序列化
func NewJSONCodec() codec.Codec {
	return codec.NewJSONCodec()
}

// NewMsgPackCodec 创建 MsgPack 编解码器
// 使用 MessagePack 格式进行消息序列化和反序列化
func NewMsgPackCodec() codec.Codec {
	return codec.NewMsgPackCodec()
}

// NewProtobufCodec 创建 Protobuf 编解码器
// 使用 Protocol Buffers 格式进行消息序列化和反序列化
func NewProtobufCodec() codec.Codec {
	return codec.NewProtobufCodec()
}
