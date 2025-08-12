package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
	wsService "github.com/tsmask/go-oam/modules/ws/service"
)

// 实例化控制层 RedisController 结构体
var NewRedis = &RedisController{
	redisService: service.NewRedis,
}

// Redis
//
// PATH /tool/redis
type RedisController struct {
	redisService *service.Redis // Redis 命令交互工具服务
}

// Redis 命令执行
//
// POST /command
//
//	@Tags			tool/Redis
//	@Accept			json
//	@Produce		json
//	@Param			command	query		string	true	"Command"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Redis run
//	@Description	Redis run
//	@Router			/tool/redis/command [post]
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
//	@Param			bindUid			query		string	true	"绑定唯一标识"						default(001)
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) Redis endpoint session
//	@Description	(ws://) Redis endpoint session
//	@Router			/tool/redis/session [get]
func (s RedisController) Session(c *gin.Context) {
	var query struct {
		BindUID string `form:"bindUid"  binding:"required"` // 绑定唯一标识
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	wsConn := ws.ServerConn{
		BindUID: query.BindUID, // 绑定唯一标识ID
	}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	go wsConn.WriteListen(1, nil)
	go wsConn.ReadListen(1, nil, s.redisService.Session)
	// 发客户端id确认是否连接
	wsService.SendOK(&wsConn, "", map[string]string{
		"clientId": wsConn.ClientId(),
	})

	// 等待停止信号
	for range wsConn.StopChan {
		wsConn.Close()
		return
	}
}
