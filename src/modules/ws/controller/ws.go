package controller

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/framework/utils/parse"
	"github.com/tsmask/go-oam/src/modules/ws/service"

	"github.com/gin-gonic/gin"
)

// NewWSController 实例化控制层 WSController 结构体
var NewWSController = &WSController{
	wsService:        service.NewWS,
	wsSendService:    service.NewWSSend,
	wsReceiveService: service.NewWSReceive,
}

// WSController WebSocket通信
//
// PATH /ws
type WSController struct {
	wsService        *service.WS        // WebSocket 服务
	wsSendService    *service.WSSend    // WebSocket消息发送 服务
	wsReceiveService *service.WSReceive // WebSocket消息接收 服务
}

// WS 通用
//
// GET /?subGroupIDs=0
//
//	@Tags			ws
//	@Accept			json
//	@Produce		json
//	@Param			subGroupID		query		string	false	"Subscribe to message groups, multiple separated by commas"
//	@Param			neUid	query		string	false	"网元唯一标识"
//	@Success		200				{object}	object	"Response Results"
//	@Summary		(ws://) Generic
//	@Description	(ws://) Generic
//	@Router			/ws [get]
func (s *WSController) WS(c *gin.Context) {
	var query struct {
		NeUID      string `form:"neUid"  binding:"required"` // 网元唯一标识
		SubGroupID string `form:"subGroupID"`                // 订阅消息组
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 订阅消息组
	var subGroupIDs []string
	if query.SubGroupID != "" {
		uniqueIDs := parse.RemoveDuplicatesToArray(query.SubGroupID, ",")
		if len(uniqueIDs) > 0 {
			subGroupIDs = uniqueIDs
		}
	}

	// 将 HTTP 连接升级为 WebSocket 连接
	conn := s.wsService.UpgraderWs(c.Writer, c.Request)
	if conn == nil {
		return
	}
	defer conn.Close()

	wsClient := s.wsService.ClientCreate(query.NeUID, subGroupIDs, conn, nil)
	go s.wsService.ClientWriteListen(wsClient)
	go s.wsService.ClientReadListen(wsClient, s.wsReceiveService.Commont)

	// 等待停止信号
	for value := range wsClient.StopChan {
		s.wsService.ClientClose(wsClient.ID)
		logger.Infof("ws Stop Client UID %s %s", wsClient.BindUid, value)
		return
	}
}

// Test 测试
//
// GET /test?clientId=xxx&groupID=xxx
func (s *WSController) Test(c *gin.Context) {
	errMsgArr := []string{}

	clientId := c.Query("clientId")
	if clientId != "" {
		err := s.wsSendService.ByClientID(c.Query("clientId"), "test message")
		if err != nil {
			errMsgArr = append(errMsgArr, "clientId: "+err.Error())
		}
	}

	groupID := c.Query("groupID")
	if groupID != "" {
		err := s.wsSendService.ByGroupID(c.Query("groupID"), "test message")
		if err != nil {
			errMsgArr = append(errMsgArr, "groupID: "+err.Error())
		}
	}

	c.JSON(200, resp.OkData(errMsgArr))
}
