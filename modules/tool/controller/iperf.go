package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/service"
)

// 实例化控制层 IPerfController 结构体
var NewIPerf = &IPerfController{
	iperfService: service.NewIPerf,
}

// iperf 网络性能测试工具 https://iperf.fr/iperf-download.php
//
// PATH /tool/iperf
type IPerfController struct {
	iperfService *service.IPerf // IPerf3 网络性能测试工具服务
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
		BindUID string `form:"bindUid"  binding:"required"` // 绑定唯一标识
		Cols    int    `form:"cols"`                        // 终端单行字符数
		Rows    int    `form:"rows"`                        // 终端显示行数
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}
	if query.Cols == 0 {
		query.Cols = 120
	}
	if query.Rows == 0 {
		query.Rows = 40
	}

	//  连接会话
	clientSession, err := cmd.NewClientSession(query.Cols, query.Rows)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	defer clientSession.Close()

	wsConn := ws.ServerConn{}
	// 将 HTTP 连接升级为 WebSocket 连接
	if err := wsConn.Upgrade(c.Writer, c.Request); err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, err.Error()))
		return
	}
	defer wsConn.Close()
	wsConn.SetAnyConn(clientSession)
	go wsConn.WriteListen(nil)
	go wsConn.ReadListen(nil, s.iperfService.Session)

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
				wsConn.SendTextJSON(fmt.Sprintf("iperf_%d", ms.UnixMilli()), resp.CODE_SUCCESS, string(outputByte), nil)
			}
		case <-wsConn.CloseSignal(): // 等待停止信号
			wsConn.Close()
			return
		}
	}
}
