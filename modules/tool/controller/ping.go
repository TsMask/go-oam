package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/cmd"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/tool/model"
	"github.com/tsmask/go-oam/modules/tool/service"
)

// NewPingController 实例化控制层 PingController 结构体
func NewPingController() *PingController {
	return &PingController{
		srv: service.NewPingService(),
	}
}

// ping ICMP网络探测工具 https://github.com/prometheus-community/pro-bing
//
// PATH /tool/ping
type PingController struct {
	srv *service.Ping // ping ICMP网络探测工具
}

// ping 基本信息运行
//
// POST /
//
//	@Tags			tool/ping
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Ping for Basic Information Running
//	@Description	Ping for Basic Information Running
//	@Router			/tool/ping [post]
func (s *PingController) Statistics(c *gin.Context) {
	var body model.Ping
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	info, err := s.srv.Statistics(body)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(info))
}

// ping 网元端版本信息
//
// GET /v
//
//	@Tags			tool/ping
//	@Summary		Ping for version information on the network element side
//	@Router			/tool/ping/v [get]
func (s *PingController) Version(c *gin.Context) {
	output, err := s.srv.Version()
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(output))
}

// ping UNIX运行
//
// GET /run
//
//	@Tags			tool/ping
//	@Summary		(ws://) Ping for UNIX runs on the network element side
//	@Router			/tool/ping/run [get]
func (s *PingController) Run(c *gin.Context) {
	var query struct {
		Cols int `form:"cols"` // 终端单行字符数
		Rows int `form:"rows"` // 终端显示行数
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

	// 连接会话
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
	go wsConn.ReadListen(nil, s.srv.Session)

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
				wsConn.SendTextJSON(fmt.Sprintf("ping_%d", ms.UnixMilli()), resp.CODE_SUCCESS, string(outputByte), nil)
			}
		case <-wsConn.CloseSignal(): // 等待停止信号
			wsConn.Close()
			return
		}
	}
}
