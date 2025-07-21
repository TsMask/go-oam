package oam

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/logger"
	"github.com/tsmask/go-oam/modules"
	"github.com/tsmask/go-oam/modules/callback"
)

// License 网元传入
type License struct {
	NeType     string
	Version    string
	SerialNum  string
	ExpiryDate string
	Capability int // AMF是GnbNum UDM是UeNum
}

// Opts SDK参数
type Opts struct {
	Dev      bool                // 开发模式
	ConfPath string              // 独立启动的配置文件路径
	License  *License            // 网元License传入
	setupArr []func(gin.IRouter) // 外部路由拓展
}

// New 初始化OAM
func New(o *Opts) *Opts {
	// 配置参数
	config.InitConfig()
	config.Set("dev", o.Dev)
	if o.License != nil {
		LicenseRrefresh(*o.License)
	}
	if o.ConfPath != "" {
		config.ReadExternalConfig(o.ConfPath)
	}
	// 程序日志
	neTypeLower := strings.ToLower(fmt.Sprint(config.Get("ne.type")))
	loggerConf := config.Get("logger").(map[string]any)
	loggerConf["filename"] = fmt.Sprintf("%s_oam.log", neTypeLower)
	logger.InitLogger()
	return o
}

// SetupCallback 相关回调功能
// 经过New初始后实现相关回调功能
func (o *Opts) SetupCallback(handler callback.CallbackHandler) {
	callback.Handler(handler)
}

// RouteExpose 在已有Gin上使用
// 经过New初始后暴露路由
func (o *Opts) RouteExpose(router gin.IRouter) error {
	defer logger.Close()
	if config.RunTime().IsZero() {
		return fmt.Errorf("config not init")
	}
	modules.RouteSetup(router)
	return nil
}

// RouteAdd 拓展装载路由
// 经过New初始后拓展装载路由
func (o *Opts) RouteAdd(setup func(gin.IRouter)) {
	if setup == nil {
		return
	}
	if len(o.setupArr) == 0 {
		o.setupArr = make([]func(gin.IRouter), 0)
	}
	o.setupArr = append(o.setupArr, setup)
}

// Run 独立运行OAM
// 经过New初始后启动OAM服务
func (o *Opts) Run() error {
	defer logger.Close()
	if config.RunTime().IsZero() {
		return fmt.Errorf("config not init")
	}
	if !config.Enable() {
		return fmt.Errorf("oam is not enable")
	}

	modules.RouteService(config.Dev(), o.setupArr)
	return nil
}

// LicenseRrefresh 刷新网元License信息
func LicenseRrefresh(lic License) {
	config.Set("ne.type", lic.NeType)
	config.Set("ne.version", lic.Version)
	config.Set("ne.serialNum", lic.SerialNum)
	config.Set("ne.expiryDate", lic.ExpiryDate)
	config.Set("ne.capability", lic.Capability)
}
