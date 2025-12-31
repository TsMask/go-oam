package tool

import (
	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/modules/tool/controller"
)

// SetupRouteIPerf iperf交互路由注册
func SetupRouteIPerf(router gin.IRouter) {
	iperf := controller.NewIPerfController()
	iperfGroup := router.Group("/tool/iperf")
	{
		iperfGroup.GET("/v", iperf.Version)
		iperfGroup.GET("/run", iperf.Run) // ws
	}
}

// SetupRoutePing ping交互路由注册
func SetupRoutePing(router gin.IRouter) {
	ping := controller.NewPingController()
	pingGroup := router.Group("/tool/ping")
	{
		pingGroup.POST("", ping.Statistics)
		pingGroup.GET("/v", ping.Version)
		pingGroup.GET("/run", ping.Run) // ws
	}
}

// SetupRouteSSH ssh交互路由注册
func SetupRouteSSH(router gin.IRouter) {
	ssh := controller.NewSSHController()
	sshGroup := router.Group("/tool/ssh")
	{
		sshGroup.POST("/command", ssh.Command)
		sshGroup.GET("/session", ssh.Session) // ws
	}
}

// SetupRouteTelnet telnet交互路由注册
func SetupRouteTelnet(router gin.IRouter) {
	telnet := controller.NewTelnetController()
	telnetGroup := router.Group("/tool/telnet")
	{
		telnetGroup.POST("/command", telnet.Command)
		telnetGroup.GET("/session", telnet.Session) // ws
	}
}

// SetupRouteSNMP snmp交互路由注册
func SetupRouteSNMP(router gin.IRouter) {
	snmp := controller.NewSNMPController()
	snmpGroup := router.Group("/tool/snmp")
	{
		snmpGroup.POST("/command", snmp.Command)
		snmpGroup.GET("/session", snmp.Session) // ws
	}
}

// SetupRouteRedis redis交互路由注册
func SetupRouteRedis(router gin.IRouter) {
	redis := controller.NewRedisController()
	redisGroup := router.Group("/tool/redis")
	{
		redisGroup.POST("/command", redis.Command)
		redisGroup.GET("/session", redis.Session) // ws
	}
}
