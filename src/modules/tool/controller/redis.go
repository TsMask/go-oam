package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/tool/service"
	wsService "github.com/tsmask/go-oam/src/modules/ws/service"
)

// 实例化控制层 RedisController 结构体
var NewRedis = &RedisController{
	redisService: service.NewRedis,
	wsService:    wsService.NewWS,
}

// Redis
//
// PATH /tool/redis
type RedisController struct {
	redisService *service.Redis // Redis 命令交互工具服务
	wsService    *wsService.WS  // WebSocket 服务
}

// Redis 命令执行
//
// GET /command
//
//	@Tags			tool/Redis
//	@Accept			json
//	@Produce		json
//	@Param			command	query		string	true	"Command"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Redis run
//	@Description	Redis run
//	@Router			/tool/redis/command [get]
func (s RedisController) Command(c *gin.Context) {
	var body struct {
		Command string `form:"command" binding:"required"` // 命令
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	output, err := s.redisService.Command(body.Command)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(output))
}

// Redis 终端会话
//
// GET /session
//
//	@Tags			tool/Redis
//	@Accept			json
//	@Produce		json
//	@Param			neUid			query		string	true	"网元唯一标识"						default(001)
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) Redis endpoint session
//	@Description	(ws://) Redis endpoint session
//	@Router			/tool/redis/session [get]
func (s RedisController) Session(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid"  binding:"required"` // 网元唯一标识
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 将 HTTP 连接升级为 WebSocket 连接
	wsConn := s.wsService.UpgraderWs(c.Writer, c.Request)
	if wsConn == nil {
		return
	}
	defer wsConn.Close()

	wsClient := s.wsService.ClientCreate(query.NeUID, nil, wsConn, nil)
	go s.wsService.ClientWriteListen(wsClient)
	go s.wsService.ClientReadListen(wsClient, s.redisService.Session)

	// 等待停止信号
	for value := range wsClient.StopChan {
		s.wsService.ClientClose(wsClient.ID)
		logger.Infof("ws Stop Client UID %s %s", wsClient.BindUid, value)
		return
	}
}
