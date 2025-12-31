package controller

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsmask/go-oam/framework/route/reqctx"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/ws/service"

	"github.com/gin-gonic/gin"
)

// NewWSController 实例化控制层 WSController 结构体
func NewWSController() *WSController {
	return &WSController{srv: service.NewWS()}
}

// WSController WebSocket通信
//
// PATH /ws
type WSController struct {
	srv *service.WS
}

// WS 通用
//
// GET /
//
//	@Tags			ws
//	@Summary		(ws://) Generic
//	@Router			/ws [get]
func (s *WSController) WS(c *gin.Context) {
	wsConn := ws.ServerConn{}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	oamCfg := reqctx.OAMConfig(c)
	wsConn.SetAnyConn(oamCfg)
	go wsConn.WriteListen(nil)
	go wsConn.ReadListen(nil, service.ReceiveCommon)

	// 记录客户端
	s.srv.ClientAdd(&wsConn)
	defer s.srv.ClientRemove(&wsConn)

	// 等待停止信号
	for range wsConn.CloseSignal() {
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
		err := s.srv.ClientSend(clientId, messageType, map[string]any{
			"msgType": msgType,
			"time":    time.Now().Format(time.RFC3339),
		})
		if err != nil {
			errMsgArr = append(errMsgArr, "clientId: "+err.Error())
		}
	}

	c.JSON(200, resp.OkData(errMsgArr))
}
