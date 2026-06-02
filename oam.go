package oam

// Option Functional Options 接口
type Option func(*OAM)

// WithVersion 设置版本配置
func WithVersion(v string) Option {
	return func(o *OAM) {
		o.version = v
	}
}

// OAM SDK 实例
type OAM struct {
	version string
}

// Version 获取当前版本
func (o *OAM) Version() string {
	return o.version
}

// New 创建 OAM 实例
func New(opts ...Option) *OAM {
	o := &OAM{}

	// 应用传入的选项
	for _, opt := range opts {
		opt(o)
	}
	return o
}
