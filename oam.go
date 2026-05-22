package oam

// OAM SDK 实例
type OAM struct {
	NE string
}

// Option Functional Options 接口
type Option func(*OAM)

// WithNEConfig 设置 NE 配置
func WithNEConfig() Option {
	return func(o *OAM) {
		o.NE = ""
	}
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
