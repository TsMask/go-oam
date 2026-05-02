package grpc

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/tsmask/go-oam/grpc/protobuf"
	"github.com/tsmask/go-oam/grpc/types"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// shardCount 分片数量
const shardCount = 256

// shardMask 分片掩码
const shardMask = shardCount - 1

// Conn 连接抽象
type Conn struct {
	ID string

	// 发送通道
	sendCh chan *types.Message

	done   chan struct{}
	closed atomic.Bool

	// 分片 pending 请求表
	shards [shardCount]struct {
		mu      sync.Mutex
		pending map[string]chan *types.Message
	}
}

func newConn(bufferSize int) *Conn {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	c := &Conn{
		ID:     newID(),
		sendCh: make(chan *types.Message, bufferSize),
		done:   make(chan struct{}),
	}

	for i := range c.shards {
		c.shards[i].pending = make(map[string]chan *types.Message)
	}

	return c
}

// Call 同步调用，等待响应
func (c *Conn) Call(ctx context.Context, action string, data []byte) ([]byte, error) {
	if c.closed.Load() {
		return nil, ErrStreamClosed
	}

	id := newID()
	msg := &types.Message{}
	msg.SetRequest(id, action, data)

	idx := int(hashString(id) & shardMask)
	respCh := make(chan *types.Message, 1)

	c.shards[idx].mu.Lock()
	c.shards[idx].pending[id] = respCh
	c.shards[idx].mu.Unlock()

	defer func() {
		c.shards[idx].mu.Lock()
		delete(c.shards[idx].pending, id)
		c.shards[idx].mu.Unlock()
	}()

	select {
	case c.sendCh <- msg:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrStreamClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respCh:
		if resp == nil {
			return nil, ErrRequestTimeout
		}
		if resp.IsError() {
			return resp.Data, &RPCError{Code: resp.Code, Msg: resp.Msg}
		}
		return resp.Data, nil
	}
}

// CallAsync 异步调用
func (c *Conn) CallAsync(ctx context.Context, action string, data []byte, callback func([]byte, error)) {
	go func() {
		resp, err := c.Call(ctx, action, data)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

// Close 关闭连接
func (c *Conn) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	close(c.done)
	close(c.sendCh)

	return nil
}

// newID 生成唯一ID
func newID() string {
	return gonanoid.Must(21)
}

func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// RPCError RPC 错误
type RPCError struct {
	Code int32
	Msg  string
}

func (e *RPCError) Error() string {
	return e.Msg
}

// newResponse 创建响应
func newResponse(id, action string, code int32, msg string, data []byte) *protobuf.Message {
	return &protobuf.Message{
		Id:     id,
		Action: action,
		Code:   code,
		Msg:    msg,
		Data:   data,
	}
}
