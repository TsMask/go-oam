package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/parse"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

// NewAlarmController 创建告警控制器
func NewAlarmController(srv *service.Alarm) *AlarmController {
	if srv == nil {
		srv = service.NewAlarm()
	}
	return &AlarmController{srv: srv}
}

// 告警
//
// PATH /alarm
type AlarmController struct {
	srv *service.Alarm
}

// 告警历史记录
//
// GET /history
//
//	@Tags			Alarm
//	@Summary		Alarm History List
//	@Router			/alarm/history [get]
func (s AlarmController) History(c *gin.Context) {
	n := parse.Number(c.Query("n"))
	data := s.srv.HistoryList(int(n))
	c.JSON(200, resp.OkData(data))
}

// 告警发送测试
//
// GET /test
//
//	@Tags			Alarm
//	@Summary		Alarm Push Test
//	@Router			/alarm/test [get]
func (s AlarmController) Test(c *gin.Context) {
	var query struct {
		NeUID string `form:"neUid" binding:"required"` // 网元唯一标识
		Url   string `form:"url" binding:"required"`   // 网管地址
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	alarmId := fmt.Sprintf("100_%d", time.Now().UnixMilli())
	addInfo := fmt.Sprintf("ClientIP: %s", c.ClientIP())
	locationInfo := fmt.Sprintf("Client UserAgent: %s", c.Request.UserAgent())
	alarm := model.Alarm{
		NeUid:             query.NeUID,                    // 网元唯一标识
		AlarmId:           alarmId,                        // 告警ID
		AlarmCode:         100,                            // 告警状态码
		AlarmType:         model.ALARM_TYPE_COMMUNICATION, // 告警类型
		AlarmTitle:        "Alarm Test",                   // 告警标题
		PerceivedSeverity: model.ALARM_SEVERITY_EVENT,     // 告警级别 Critical,Major,Minor,Warning,Event
		AlarmStatus:       model.ALARM_STATUS_CLEAR,       // 告警状态 Clear,Active
		SpecificProblem:   "Alarm Test",                   // 告警问题原因
		SpecificProblemID: "100",                          // 告警问题原因ID
		AddInfo:           addInfo,                        // 告警辅助信息
		LocationInfo:      locationInfo,                   // 告警定位信息
	}
	err := s.srv.PushURL(query.Url, &alarm, 1*time.Minute)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
