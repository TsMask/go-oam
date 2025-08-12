package oam

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/modules"
	"github.com/tsmask/go-oam/modules/callback"
)

// License 网元传入
type License struct {
	NeType     string // 网元类型 大写
	Version    string // 版本号 格式：X.Y.Z
	SerialNum  string // 序列号 8位字符
	ExpiryDate string // 有效日期 格式：YYYY-MM-DD
	NbNumber   int    // 基站限制数量，AMF MME
	UeNumber   int    // 终端限制数量 UDM
}

// Listen 路由HTTP服务监听配置
type Listen struct {
	Addr   string // 监听地址 格式：ip:port
	Schema string // 监听协议 http/https
	Cert   string // 证书文件路径，仅https协议需要
	Key    string // 私钥文件路径，仅https协议需要
}

// Upload 文件上传配置
type Upload struct {
	FileDir   string   // 文件上传目录路径，默认：/tmp
	FileSize  int      // 最大上传文件大小，单位MB，默认：1
	Whitelist []string // 文件扩展名白名单
}

// Opts SDK参数
type Opts struct {
	License   License             // 网元License传入
	ListenArr []Listen            // 启动的监听地址
	Upload    Upload              // 文件上传配置
	setupArr  []func(gin.IRouter) // 外部路由拓展
}

// New 初始化OAM
func New(o *Opts) *Opts {
	// 配置参数
	config.InitConfig()
	LicenseRrefresh(o.License)
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
	if config.RunTime().IsZero() {
		return fmt.Errorf("[OAM] config not init")
	}
	// 启动的监听地址
	if o.ListenArr != nil {
		listenArr := make([]any, 0)
		for _, v := range o.ListenArr {
			item := map[string]any{
				"addr":   v.Addr,
				"schema": v.Schema,
				"cert":   v.Cert,
				"key":    v.Key,
			}
			listenArr = append(listenArr, item)
		}
		config.Set("route", listenArr)
	}
	return modules.RouteService(o.setupArr)
}

// LicenseRrefresh 刷新网元License信息
func LicenseRrefresh(lic License) {
	neConf, ok := config.Get("ne").(map[string]any)
	if !ok {
		return
	}
	neConf["type"] = lic.NeType
	neConf["version"] = lic.Version
	neConf["serialnum"] = lic.SerialNum
	neConf["expirydate"] = lic.ExpiryDate
	neConf["nbnumber"] = lic.NbNumber
	neConf["uenumber"] = lic.UeNumber
}

// ConfigUpload 上传文件配置
func ConfigUpload(uploadConfig Upload) {
	upload, ok := config.Get("upload").(map[string]any)
	if !ok {
		return
	}
	upload["filedir"] = uploadConfig.FileDir
	upload["filesize"] = uploadConfig.FileSize
	list := make([]any, 0)
	for _, v := range uploadConfig.Whitelist {
		list = append(list, v)
	}
	upload["whitelist"] = list
}
