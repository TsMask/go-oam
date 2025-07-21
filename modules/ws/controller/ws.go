package controller

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/logger"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/ws/service"

	"github.com/gin-gonic/gin"
)

// NewWSController 实例化控制层 WSController 结构体
var NewWSController = &WSController{}

// WSController WebSocket通信
//
// PATH /ws
type WSController struct{}

// WS 通用
//
// GET /
//
//	@Tags			ws
//	@Accept			json
//	@Produce		json
//	@Param			neUid	query		string	false	"网元唯一标识"
//	@Success		200				{object}	object	"Response Results"
//	@Summary		(ws://) Generic
//	@Description	(ws://) Generic
//	@Router			/ws [get]
func (s *WSController) WS(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid"  binding:"required"` // 网元唯一标识
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	wsConn := ws.ServerConn{
		BindUID: query.NeUID, // 绑定唯一标识ID
	}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	go wsConn.WriteListen(1, func(err error) {
		logger.Errorf("ws WriteListen err: %s", err.Error())
	})
	go wsConn.ReadListen(1, func(err error) {
		logger.Errorf("ws ReadListen err: %s", err.Error())
	}, service.ReceiveCommont)
	// 发客户端id确认是否连接
	wsConn.SendJSON(resp.OkData(map[string]string{
		"clientId": wsConn.ClientId(),
	}))

	// 记录客户端
	service.ClientAdd(&wsConn)
	defer service.ClientRemove(&wsConn)

	// 等待停止信号
	for value := range wsConn.StopChan {
		wsConn.Close()
		logger.Infof("ws Stop Client UID %s %s", wsConn.BindUID, value)
		return
	}
}

// Test 测试
//
// GET /test?clientId=xxx
func (s *WSController) Test(c *gin.Context) {
	errMsgArr := []string{}

	clientId := c.Query("clientId")
	if clientId != "" {
		err := service.ClientSend(c.Query("clientId"), "test message")
		if err != nil {
			errMsgArr = append(errMsgArr, "clientId: "+err.Error())
		}
	}

	c.JSON(200, resp.OkData(errMsgArr))
}
