package modules

import (
	"github.com/tsmask/go-oam/src/framework/route"

	"github.com/tsmask/go-oam/src/modules/common"
	"github.com/tsmask/go-oam/src/modules/push"
	"github.com/tsmask/go-oam/src/modules/state"
	"github.com/tsmask/go-oam/src/modules/tool"
	"github.com/tsmask/go-oam/src/modules/ws"

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
func RouteService(setupArr []func(gin.IRouter)) {
	router := route.Engine()
	// 装载外部拓展
	if len(setupArr) > 0 {
		for _, setup := range setupArr {
			setup(router)
		}
	}
	RouteSetup(router)
	route.Run(router)
}
