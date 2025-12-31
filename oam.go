package oam

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/modules"
	"github.com/tsmask/go-oam/modules/callback"
)

// OAM SDK 实例
type OAM struct {
	cfg      *config.Config
	handler  callback.CallbackHandler
	setupArr []func(gin.IRouter)
	Push     *Push
}

// Option Functional Options 接口
type Option func(*OAM)

// WithNEConfig 设置 NE 配置
func WithNEConfig(neCfg config.NEConfig) Option {
	return func(o *OAM) {
		o.cfg.Update(func(c *config.Config) {
			c.NE = neCfg
		})
	}
}

// WithRouteConfig 设置路由配置
func WithRouteConfig(routes []config.RouteConfig) Option {
	return func(o *OAM) {
		if len(routes) > 0 {
			o.cfg.Update(func(c *config.Config) {
				c.Route = routes
			})
		}
	}
}

// WithUploadConfig 设置文件上传配置
func WithUploadConfig(uploadCfg config.UploadConfig) Option {
	return func(o *OAM) {
		o.cfg.Update(func(c *config.Config) {
			c.Upload = uploadCfg
		})
	}
}

// WithOMCConfig 设置 OMC 配置
func WithOMCConfig(omcCfg config.OMCConfig) Option {
	return func(o *OAM) {
		o.cfg.Update(func(c *config.Config) {
			c.OMC = omcCfg
		})
	}
}

// WithExternalConfig 从外部文件加载配置
func WithExternalConfig(configPath string) Option {
	return func(o *OAM) {
		extC, err := config.LoadExternalConfig(configPath)
		if err != nil {
			fmt.Printf("[OAM] load external config error: %s\n", err.Error())
			return
		}
		o.cfg.Merge(extC)
	}
}

// WithCallbackHandler 设置回调处理器
func WithCallbackHandler(handler callback.CallbackHandler) Option {
	return func(o *OAM) {
		o.handler = handler
	}
}

// WithPush 开启推送功能
func WithPush() Option {
	return func(o *OAM) {
		o.Push = NewPush(o)
	}
}

// New 创建 OAM 实例
func New(opts ...Option) *OAM {
	o := &OAM{
		cfg:      config.New(),
		handler:  &callback.CallbackFuncs{},
		setupArr: make([]func(gin.IRouter), 0),
	}

	// 应用传入的选项
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// SetupPushRoute 注册推送相关路由
func (o *OAM) SetupPushRoute(router gin.IRouter) {
	if o.Push == nil {
		o.Push = NewPush(o)
	}
	o.Push.SetupRoute(router)
}

// SetupCallback 相关回调功能
func (o *OAM) SetupCallback(handler callback.CallbackHandler) {
	o.handler = handler
}

// SetupRoute 拓展装载路由, 可以装载多次
func (o *OAM) SetupRoute(setup func(gin.IRouter)) {
	o.setupArr = append(o.setupArr, setup)
}

// RouteEngine 路由引擎, 已有Gin进行装载路由
func (o *OAM) RouteEngine(router *gin.Engine) *gin.Engine {
	router.Use(reqctx.ConfigInContext(o.cfg), reqctx.CallbackInContext(o.handler))
	for _, setup := range o.setupArr {
		setup(router)
	}
	return router
}

// Run 启动 OAM 服务
func (o *OAM) Run() error {
	router := modules.NewRouter()
	o.RouteEngine(router)
	return modules.RunServer(o.cfg, router)
}

// GetConfig 获取配置实例
func (o *OAM) GetConfig() *config.Config {
	return o.cfg
}
