# OAM SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/olekukonko/tablewriter.svg)](https://pkg.go.dev/github.com/tsmask/go-oam)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsmask/go-oam)](https://goreportcard.com/report/github.com/tsmask/go-oam)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg)](LICENSE)
[![Tag](https://img.shields.io/badge/TAG-list-success)](https://proxy.golang.org/github.com/tsmask/go-oam/@v/list)

## 简介

OAM SDK 是一个用于网元与网管进行交互的函数集

## 功能模块

### 通用模块

- 本地文件操作（读取列表，下载，删除）
- 上传文件操作（上传，分片上传）
- 服务器时间

### 工具模块

- SSH 命令下发
- Telnet 命令下发（需网元提供回调）
- redis 命令下发（需网元提供回调）
- ping 命令下发（需安装命令程序）
- iperf 命令下发（需安装命令程序）

### WS 模块

- WebSocket 连接

### 状态模块

- 网元状态
- 系统状态
- 机器资源状态

### 上报模块

- 终端接入基站
  1. 上报推送，接收回调
  2. 历史查询，数据集最大4096条
- KPI
  1. 上报推送，接收回调
  2. 历史查询，数据集最大4096条
- 告警
  1. 上报推送，接收回调
  2. 历史查询，数据集最大4096条
- 话单
  1. 上报推送，接收回调
  2. 历史查询，数据集最大4096条
- 基站状态
  1. 上报推送，接收回调
  2. 历史查询，数据集最大4096条
- 通用
  1. 上报推送，接收回调
  2. 历史查询，数据集最大4096条

### 下发模块

- 网管信息
- 网元配置（需网元提供回调）

## 使用方法

1. 下载完整依赖库，在`go.mod`文件中引入替换为本地库

```mod
replace github.com/tsmask/go-oam v1.0.0 => ./lib/go-oam

require (
	github.com/tsmask/go-oam v1.0.0
)
```

2. 已有 Gin 上使用代码

```go
// 导入库
import "github.com/tsmask/go-oam"


// oamCallback 回调功能
type oamCallback struct{}
// Standby implements callback.CallbackHandler.
func (o *oamCallback) Standby() bool {
	return false
}
// Redis implements callback.CallbackHandler.
func (o *oamCallback) Redis() any {
	// *redis.Client
	return nil
}
// Telent implements callback.CallbackHandler.
func (o *oamCallback) Telent(command string) string {
	return "Telent implements"
}
// SNMP implements callback.CallbackHandler.
func (o *oamCallback) SNMP(oid, operType string, value any) any {
	return "SNMP implements"
}
// Config implements callback.CallbackHandler.
func (o *oamCallback) Config(action, paramName, loc string, paramValue any) error {
	return fmt.Errorf("config => %s > %s > %s > %v", action, paramName, loc, paramValue)
}

// 加入OAM相关接口模块
o := oam.New(&oam.Opts{
  License: oam.License{
    NeType:     "NE",
    Version:    "1.0",
    SerialNum:  "1234567890",
    ExpiryDate: "2025-12-31",
    NbNumber:   10,
    UeNumber:   100,
  },
})
o.SetupCallback(new(oamCallback))
if err := o.RouteExpose(r); err != nil {
  fmt.Printf("oam run fail: %s\n", err.Error())
}

```

## 目录结构

```text
go-oam
├── .vscode                               目录-vscode配置
├── dev                                   目录-本地开发配置
│   ├── certs                             目录-证书
│   └── oam.yaml                          文件-配置文件
├── examples                              目录-运行示例
│   ├── independent_coprocessor           目录-独立协程模式
│   ├── oam_manager                       目录-现有模式
│   └── standalone                        目录-独立模式
├── framework                             目录-核心框架
│   ├── cmd                               目录-本地命令行
│   ├── config                            目录-配置文件
│   ├── fetch                             目录-网络请求封装
│   ├── router                            目录-gin路由引擎
│   └── ...
├── modules                               目录-模块
│   ├── callback                          目录-回调处理
│   ├── common                            目录-通用模块
│   ├── push                              目录-推送模块
│   ├── state                             目录-状态模块
│   ├── tool                              目录-工具模块
│   ├── ws                                目录-WS模块
│   └── modules.go                        文件-加载模块
├── oam_pull_omc.go                       文件-下发函数OMC
├── oam_push_alarm.go                     文件-推送函数告警
├── oam_push_cdr.go                       文件-推送函数话单
├── oam_push_common.go                    文件-推送函数通用
├── oam_push_kpi.go                       文件-推送函数KPI
├── oam_push_nb_state.go                  文件-推送函数基站状态
├── oam_push_ue_nb.go                     文件-推送函数UENB
├── oam.go                                文件-库函数
├── LICENSE                               文件-许可证
└──  README.md                            文件-项目说明
```
