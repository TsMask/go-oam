package ws

import "github.com/tsmask/go-oam/ws/codec"

// NewJSONCodec 创建 JSON 编解码器
func NewJSONCodec() codec.Codec { return codec.NewJSONCodec() }

// NewMsgPackCodec 创建 MessagePack 编解码器
func NewMsgPackCodec() codec.Codec { return codec.NewMsgPackCodec() }

// NewProtobufCodec 创建 Protobuf 编解码器
func NewProtobufCodec() codec.Codec { return codec.NewProtobufCodec() }