package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/src/framework/cmd"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/modules/tool/service"
	wsService "github.com/tsmask/go-oam/src/modules/ws/service"
)

// 实例化控制层 IPerfController 结构体
var NewIPerf = &IPerfController{
	iperfService: service.NewIPerf,
	wsService:    wsService.NewWS,
}

// iperf 网络性能测试工具 https://iperf.fr/iperf-download.php
//
// PATH /tool/iperf
type IPerfController struct {
	iperfService *service.IPerf // IPerf3 网络性能测试工具服务
	wsService    *wsService.WS  // WebSocket 服务
}

// iperf 版本信息
//
// GET /v
//
//	@Tags			tool/iperf
//	@Accept			json
//	@Produce		json
//	@Param			version	query		string	true	"Version"	Enums(V2, V3)
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		iperf version information
//	@Description	iperf version information
//	@Router			/tool/iperf/v [get]
func (s IPerfController) Version(c *gin.Context) {
	var query struct {
		Version string `form:"version" binding:"required,oneof=V2 V3"` // 版本
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	output, err := s.iperfService.Version(query.Version)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	data := strings.Split(output, "\n")
	c.JSON(200, resp.OkData(data))
}

// iperf 软件运行
//
// GET /run
//
//	@Tags			tool/iperf
//	@Accept			json
//	@Produce		json
//	@Param			neType			query		string	true	"NE Type"					Enums(IMS,AMF,AUSF,UDM,SMF,PCF,NSSF,NRF,UPF,MME,CBC,oam,SGWC,SMSC)
//	@Param			neId			query		string	true	"NE ID"						default(001)
//	@Param			cols			query		number	false	"Terminal line characters"	default(120)
//	@Param			rows			query		number	false	"Terminal display lines"	default(40)
//	@Param			access_token	query		string	true	"Authorization"
//	@Success		200				{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		(ws://) iperf software running
//	@Description	(ws://) iperf software running
//	@Router			/tool/iperf/run [get]
func (s IPerfController) Run(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid"  binding:"required"` // 网元唯一标识
		Cols  int    `form:"cols"`                      // 终端单行字符数
		Rows  int    `form:"rows"`                      // 终端显示行数
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	//  连接会话
	clientSession, err := cmd.NewClientSession(query.Cols, query.Rows)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	defer clientSession.Close()

	// 将 HTTP 连接升级为 WebSocket 连接
	wsConn := s.wsService.UpgraderWs(c.Writer, c.Request)
	if wsConn == nil {
		return
	}
	defer wsConn.Close()

	wsClient := s.wsService.ClientCreate(query.NeUID, nil, wsConn, clientSession)
	go s.wsService.ClientWriteListen(wsClient)
	go s.wsService.ClientReadListen(wsClient, s.iperfService.Run)

	// 等待1秒，排空首次消息
	time.Sleep(1 * time.Second)
	_ = clientSession.Read()

	// 实时读取Run消息直接输出
	msTicker := time.NewTicker(100 * time.Millisecond)
	defer msTicker.Stop()
	for {
		select {
		case ms := <-msTicker.C:
			outputByte := clientSession.Read()
			if len(outputByte) > 0 {
				outputStr := string(outputByte)
				msgByte, _ := json.Marshal(resp.Ok(map[string]any{
					"requestId": fmt.Sprintf("iperf3_%d", ms.UnixMilli()),
					"data":      outputStr,
				}))
				wsClient.MsgChan <- msgByte
			}
		case <-wsClient.StopChan: // 等待停止信号
			s.wsService.ClientClose(wsClient.ID)
			logger.Infof("ws Stop Client UID %s", wsClient.BindUid)
			return
		}
	}
}
