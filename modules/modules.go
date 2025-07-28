package modules

import (
	"github.com/tsmask/go-oam/framework/route"

	"github.com/tsmask/go-oam/modules/common"
	"github.com/tsmask/go-oam/modules/push"
	"github.com/tsmask/go-oam/modules/state"
	"github.com/tsmask/go-oam/modules/tool"
	"github.com/tsmask/go-oam/modules/ws"

	"github.com/gin-gonic/gin"
)

// RouteSetup 路由装载，加入已有GIn
func RouteSetup(router gin.IRouter) {
	// 通用模块
	common.SetupRoute(router)
	// 工具模块
	tool.SetupRoute(router)
	// ws 模块
	ws.SetupRoute(router)
	// 状态模块
	state.SetupRoute(router)
	// 上报模块
	push.SetupRoute(router)
}

// RouteService 路由独立服务启动
func RouteService(dev bool, setupArr []func(gin.IRouter)) error {
	router := route.Engine(dev)
	// 装载外部拓展
	if len(setupArr) > 0 {
		for _, setup := range setupArr {
			setup(router)
		}
	}
	// 路由装载
	RouteSetup(router)
	return route.Run(router)
}
