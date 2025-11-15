package controller

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
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
//	@Param			bindUid	query		string	false	"绑定唯一标识"
//	@Success		200				{object}	object	"Response Results"
//	@Summary		(ws://) Generic
//	@Description	(ws://) Generic
//	@Router			/ws [get]
func (s *WSController) WS(c *gin.Context) {
	var query struct {
		BindUID string `form:"bindUid"  binding:"required"`                   // 绑定唯一标识
		MsgType string `form:"msgType"  binding:"required,oneof=text binary"` // 消息类型
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	wsConn := ws.ServerConn{BindUID: query.BindUID}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	go wsConn.WriteListen(nil)
	go wsConn.ReadListen(nil, service.ReceiveCommon)
	// 发客户端id确认是否连接
	wsConn.SendTextJSON("", resp.CODE_SUCCESS, resp.MSG_SUCCCESS, map[string]string{
		"clientId": wsConn.ClientId(),
	})

	// 记录客户端
	service.ClientAdd(&wsConn)
	defer service.ClientRemove(&wsConn)

	// 等待停止信号
	for range wsConn.StopChan {
		wsConn.Close()
		return
	}
}

// Test 测试
//
// GET /test?clientId=xxx
func (s *WSController) Test(c *gin.Context) {
	errMsgArr := []string{}

	clientId := c.Query("clientId")
	msgType := c.DefaultQuery("msgType", "text")
	messageType := websocket.TextMessage
	if msgType == "binary" {
		messageType = websocket.BinaryMessage
	}
	if clientId != "" {
		err := service.ClientSend(clientId, messageType, map[string]string{
			"msgType": msgType,
			"time":    time.Now().Format(time.RFC3339),
		})
		if err != nil {
			errMsgArr = append(errMsgArr, "clientId: "+err.Error())
		}
	}

	c.JSON(200, resp.OkData(errMsgArr))
}
