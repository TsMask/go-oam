# OAM SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/olekukonko/tablewriter.svg)](https://pkg.go.dev/github.com/tsmask/go-oam)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsmask/go-oam)](https://goreportcard.com/report/github.com/tsmask/go-oam)
[![License](https://img.shields.io/badge/license-AGPL3-blue.svg)](LICENSE)
[![Tag](https://img.shields.io/badge/TAG-list-success)](https://proxy.golang.org/github.com/tsmask/go-oam/@v/list)

## 简介

OAM SDK 是一个用于网元与网管进行交互的函数集

## 模块

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
  2. 历史查询，仅一小时数据
- KPI
  1. 上报推送，接收回调
  2. 历史查询，仅一小时数据
- 告警
  1. 上报推送，接收回调
  2. 历史查询，仅当日数据
- 话单
  1. 上报推送，接收回调
  2. 历史查询，仅十分钟数据
- 基站状态
  1. 上报推送，接收回调
  2. 历史查询，仅一小时数据
- 通用
  1. 上报推送，接收回调
  2. 历史查询，自定义清除时间

## 使用方法

1. 下载完整依赖库，在`go.mod`文件中引入

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
func (o *oamCallback) SNMP(command string) string {
	return "SNMP implements"
}


// 加入OAM相关接口模块
o := oam.New(&oam.Opts{
    License: &oam.License{
        NeType:     "NE",
        Version:    "1.0",
        SerialNum:  "1234567890",
        ExpiryDate: "2025-12-31",
        Capability: 100,
    },
})
o.SetupCallback(new(oamCallback))
if err := o.RouteExpose(router); err != nil {
    fmt.Printf("oam run fail: %s\n", err.Error())
}

```
