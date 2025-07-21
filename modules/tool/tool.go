package tool

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/logger"
	"github.com/tsmask/go-oam/modules/tool/controller"
)

// 模块路由注册
func SetupRoute(router gin.IRouter) {
	logger.Infof("开始加载 ====> tool 模块路由")

	// iperf 网络性能测试工具
	toolIperf := controller.NewIPerf
	iperfGroup := router.Group("/tool/iperf")
	{
		iperfGroup.GET("/v", toolIperf.Version)
		iperfGroup.GET("/run", toolIperf.Run) // ws
	}

	// ping ICMP网络探测工具
	toolPing := controller.NewPing
	pingGroup := router.Group("/tool/ping")
	{
		pingGroup.POST("", toolPing.Statistics)
		pingGroup.GET("/v", toolPing.Version)
		pingGroup.GET("/run", toolPing.Run) // ws
	}

	// ssh 终端命令交互工具
	toolSSH := controller.NewSSH
	sshGroup := router.Group("/tool/ssh")
	{
		sshGroup.POST("/command", toolSSH.Command)
		sshGroup.GET("/session", toolSSH.Session) // ws
	}

	// telnet 命令交互工具
	toolTelnet := controller.NewTelnet
	telnetGroup := router.Group("/tool/telnet")
	{
		telnetGroup.POST("/command", toolTelnet.Command)
		telnetGroup.GET("/session", toolTelnet.Session) // ws
	}

	// snmp 命令交互工具
	toolSnmp := controller.NewTelnet
	snmpGroup := router.Group("/tool/snmp")
	{
		snmpGroup.POST("/command", toolSnmp.Command)
		snmpGroup.GET("/session", toolSnmp.Session) // ws
	}

	// redis 命令交互工具
	toolRedis := controller.NewRedis
	redisGroup := router.Group("/tool/redis")
	{
		redisGroup.POST("/command", toolRedis.Command)
		redisGroup.GET("/session", toolRedis.Session) // ws
	}
}
