package core

import (
	"sync"
)

// Pool 内存对象池
// 复用byte数组，减少GC压力
type buffer struct {
	data []byte
}

// Pool 分级内存池
// 根据大小分配到不同层级的pool
type Pool struct {
	small  sync.Pool // < 256 bytes
	medium sync.Pool // 256-1024 bytes
	large  sync.Pool // > 1024 bytes
}

// NewPool 创建内存池
func NewPool() *Pool {
	return &Pool{
		small: sync.Pool{
			New: func() any {
				return &buffer{data: make([]byte, 128)}
			},
		},
		medium: sync.Pool{
			New: func() any {
				return &buffer{data: make([]byte, 512)}
			},
		},
		large: sync.Pool{
			New: func() any {
				return &buffer{data: make([]byte, 2048)}
			},
		},
	}
}

// Get 从池中获取buffer
// 参数：size 需要的buffer大小
// 返回：byte数组
func (p *Pool) Get(size int) []byte {
	var b *buffer
	switch {
	case size <= 128:
		v := p.small.Get()
		b = v.(*buffer)
	case size <= 512:
		v := p.medium.Get()
		b = v.(*buffer)
	default:
		v := p.large.Get()
		b = v.(*buffer)
	}

	if cap(b.data) < size {
		b.data = make([]byte, size)
	}
	return b.data[:size]
}

// Put 归还buffer到池中
// 参数：buf 要归还的buffer
func (p *Pool) Put(buf []byte) {
	switch cap(buf) {
	case 128:
		b := &buffer{data: buf[:0]}
		p.small.Put(b)
	case 512:
		b := &buffer{data: buf[:0]}
		p.medium.Put(b)
	}
}
