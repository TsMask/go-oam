package oam

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/modules/push"
	"github.com/tsmask/go-oam/modules/push/model"
	"github.com/tsmask/go-oam/modules/push/service"
)

type (
	Alarm   = model.Alarm
	CDR     = model.CDR
	Common  = model.Common
	KPI     = model.KPI
	NBState = model.NBState
	UENB    = model.UENB
)

const (
	ALARM_TYPE_COMMUNICATION_ALARM = model.ALARM_TYPE_COMMUNICATION_ALARM
	ALARM_TYPE_EQUIPMENT_ALARM     = model.ALARM_TYPE_EQUIPMENT_ALARM
	ALARM_TYPE_PROCESSING_FAILURE  = model.ALARM_TYPE_PROCESSING_FAILURE
	ALARM_TYPE_ENVIRONMENTAL_ALARM = model.ALARM_TYPE_ENVIRONMENTAL_ALARM
	ALARM_TYPE_QUALITY_OF_SERVICE  = model.ALARM_TYPE_QUALITY_OF_SERVICE_ALARM

	ALARM_SEVERITY_CRITICAL = model.ALARM_SEVERITY_CRITICAL
	ALARM_SEVERITY_MAJOR    = model.ALARM_SEVERITY_MAJOR
	ALARM_SEVERITY_MINOR    = model.ALARM_SEVERITY_MINOR
	ALARM_SEVERITY_WARNING  = model.ALARM_SEVERITY_WARNING
	ALARM_SEVERITY_EVENT    = model.ALARM_SEVERITY_EVENT

	ALARM_STATUS_CLEAR  = model.ALARM_STATUS_CLEAR
	ALARM_STATUS_ACTIVE = model.ALARM_STATUS_ACTIVE

	NB_STATE_ON  = model.NB_STATE_ON
	NB_STATE_OFF = model.NB_STATE_OFF

	UENB_TYPE_AUTH   = model.UENB_TYPE_AUTH
	UENB_TYPE_DETACH = model.UENB_TYPE_DETACH
	UENB_TYPE_CM     = model.UENB_TYPE_CM

	UENB_RESULT_AUTH_SUCCESS                            = model.UENB_RESULT_AUTH_SUCCESS
	UENB_RESULT_AUTH_NETWORK_FAILURE                    = model.UENB_RESULT_AUTH_NETWORK_FAILURE
	UENB_RESULT_AUTH_INTERFACE_FAILURE                  = model.UENB_RESULT_AUTH_INTERFACE_FAILURE
	UENB_RESULT_AUTH_MAC_FAILURE                        = model.UENB_RESULT_AUTH_MAC_FAILURE
	UENB_RESULT_AUTH_SYNC_FAILURE                       = model.UENB_RESULT_AUTH_SYNC_FAILURE
	UENB_RESULT_AUTH_NON_5G_AUTHENTICATION_NOT_ACCEPTED = model.UENB_RESULT_AUTH_NON_5G_AUTHENTICATION_NOT_ACCEPTED
	UENB_RESULT_AUTH_RESPONSE_FAILURE                   = model.UENB_RESULT_AUTH_RESPONSE_FAILURE
	UENB_RESULT_AUTH_UNKNOWN                            = model.UENB_RESULT_AUTH_UNKNOWN
	UENB_RESULT_CM_CONNECTED                            = model.UENB_RESULT_CM_CONNECTED
	UENB_RESULT_CM_IDLE                                 = model.UENB_RESULT_CM_IDLE
	UENB_RESULT_CM_INACTIVE                             = model.UENB_RESULT_CM_INACTIVE

	ALARM_PUSH_URI    = service.ALARM_PUSH_URI
	CDR_PUSH_URI      = service.CDR_PUSH_URI
	COMMON_PUSH_URI   = service.COMMON_PUSH_URI
	KPI_PUSH_URI      = service.KPI_PUSH_URI
	NB_STATE_PUSH_URI = service.NB_STATE_PUSH_URI
	UENB_PUSH_URI     = service.UENB_PUSH_URI
)

// Push 推送功能集
type Push struct {
	o  *OAM
	mu sync.RWMutex

	alarmSrv   *service.Alarm
	cdrSrv     *service.CDR
	commonSrv  *service.Common
	kpiSrv     *service.KPI
	nbStateSrv *service.NBState
	uenbSrv    *service.UENB
}

// NewPush 创建推送功能集
func NewPush(o *OAM) *Push {
	return &Push{
		o: o,
	}
}

func (p *Push) getAlarmSrv() *service.Alarm {
	p.mu.RLock()
	if p.alarmSrv != nil {
		defer p.mu.RUnlock()
		return p.alarmSrv
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.alarmSrv == nil {
		p.alarmSrv = service.NewAlarm()
	}
	return p.alarmSrv
}

func (p *Push) getCDRSrv() *service.CDR {
	p.mu.RLock()
	if p.cdrSrv != nil {
		defer p.mu.RUnlock()
		return p.cdrSrv
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cdrSrv == nil {
		p.cdrSrv = service.NewCDR()
	}
	return p.cdrSrv
}

func (p *Push) getCommonSrv() *service.Common {
	p.mu.RLock()
	if p.commonSrv != nil {
		defer p.mu.RUnlock()
		return p.commonSrv
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.commonSrv == nil {
		p.commonSrv = service.NewCommon()
	}
	return p.commonSrv
}

func (p *Push) getKPISrv() *service.KPI {
	p.mu.RLock()
	if p.kpiSrv != nil {
		defer p.mu.RUnlock()
		return p.kpiSrv
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.kpiSrv == nil {
		var neUid string
		var kpiGranularity int
		var baseURL string
		p.o.cfg.View(func(c *config.Config) {
			neUid = c.OMC.NeUID
			kpiGranularity = c.OMC.KPIGranularity
			baseURL = c.OMC.URL
		})
		p.kpiSrv = service.NewKPI(neUid, time.Duration(kpiGranularity)*time.Second)

		// 如果配置了周期且有基础 URL，则自动启动定时推送
		if kpiGranularity > 0 && baseURL != "" {
			url := p.joinURL(baseURL, service.KPI_PUSH_URI)
			p.kpiSrv.KPITimerStart(url)
		}
	}
	return p.kpiSrv
}

func (p *Push) getNBStateSrv() *service.NBState {
	p.mu.RLock()
	if p.nbStateSrv != nil {
		defer p.mu.RUnlock()
		return p.nbStateSrv
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.nbStateSrv == nil {
		p.nbStateSrv = service.NewNBState()
	}
	return p.nbStateSrv
}

func (p *Push) getUENBSrv() *service.UENB {
	p.mu.RLock()
	if p.uenbSrv != nil {
		defer p.mu.RUnlock()
		return p.uenbSrv
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.uenbSrv == nil {
		p.uenbSrv = service.NewUENB()
	}
	return p.uenbSrv
}

// joinURL 安全拼接 URL
func (p *Push) joinURL(baseURL, uri string) string {
	u, err := url.JoinPath(baseURL, uri)
	if err != nil {
		return ""
	}
	return u
}

// SetupRoute 注册路由
func (p *Push) SetupRoute(router gin.IRouter) {
	// 路由注册会自动按需初始化 Service
	push.SetupRouteAlarm(router, p.getAlarmSrv())
	push.SetupRouteCDR(router, p.getCDRSrv())
	push.SetupRouteCommon(router, p.getCommonSrv())
	push.SetupRouteKPI(router, p.getKPISrv())
	push.SetupRouteNBState(router, p.getNBStateSrv())
	push.SetupRouteUENB(router, p.getUENBSrv())
}

// Alarm 推送告警
func (p *Push) Alarm(alarm *model.Alarm) error {
	var baseURL, neUid string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
		neUid = c.OMC.NeUID
	})

	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}

	if alarm.NeUid == "" {
		alarm.NeUid = neUid
	}

	url := p.joinURL(baseURL, service.ALARM_PUSH_URI)
	return p.getAlarmSrv().PushURL(url, alarm, 0)
}

// AlarmURL 推送告警自定义URL
func (p *Push) AlarmURL(url string, alarm *model.Alarm, timeout time.Duration) error {
	var neUid string
	p.o.cfg.View(func(c *config.Config) {
		neUid = c.OMC.NeUID
	})
	if alarm.NeUid == "" {
		alarm.NeUid = neUid
	}
	return p.getAlarmSrv().PushURL(url, alarm, timeout)
}

// AlarmHistoryList 获取告警推送历史
func (p *Push) AlarmHistoryList(n int) []model.Alarm {
	return p.getAlarmSrv().HistoryList(n)
}

// AlarmHistorySetSize 设置告警推送历史大小
func (p *Push) AlarmHistorySetSize(size int) {
	p.getAlarmSrv().HistorySetSize(size)
}

// Common 推送通用数据
func (p *Push) Common(common *model.Common) error {
	var baseURL, neUid string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
		neUid = c.OMC.NeUID
	})

	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}

	if common.NeUid == "" {
		common.NeUid = neUid
	}

	url := p.joinURL(baseURL, service.COMMON_PUSH_URI)
	return p.getCommonSrv().PushURL(url, common, time.Hour)
}

// CommonURL 推送通用数据自定义URL
func (p *Push) CommonURL(url string, common *model.Common, timeout time.Duration) error {
	var neUid string
	p.o.cfg.View(func(c *config.Config) {
		neUid = c.OMC.NeUID
	})
	if common.NeUid == "" {
		common.NeUid = neUid
	}
	return p.getCommonSrv().PushURL(url, common, timeout)
}

// CommonHistoryList 获取通用推送历史
func (p *Push) CommonHistoryList(typeStr string, n int) []model.Common {
	return p.getCommonSrv().HistoryList(typeStr, n)
}

// CommonHistorySetSize 设置通用推送历史大小
func (p *Push) CommonHistorySetSize(size int) {
	p.getCommonSrv().HistorySetSize(size)
}

// UENB 推送终端接入基站
func (p *Push) UENB(uenb *model.UENB) error {
	var baseURL, neUid string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
		neUid = c.OMC.NeUID
	})

	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}

	if uenb.NeUid == "" {
		uenb.NeUid = neUid
	}

	url := p.joinURL(baseURL, service.UENB_PUSH_URI)
	return p.getUENBSrv().PushURL(url, uenb, 0)
}

// UENBURL 推送终端接入基站自定义URL
func (p *Push) UENBURL(url string, uenb *model.UENB, timeout time.Duration) error {
	var neUid string
	p.o.cfg.View(func(c *config.Config) {
		neUid = c.OMC.NeUID
	})
	if uenb.NeUid == "" {
		uenb.NeUid = neUid
	}
	return p.getUENBSrv().PushURL(url, uenb, timeout)
}

// UENBHistoryList 获取终端接入推送历史
func (p *Push) UENBHistoryList(n int) []model.UENB {
	return p.getUENBSrv().HistoryList(n)
}

// UENBHistorySetSize 设置终端接入推送历史大小
func (p *Push) UENBHistorySetSize(size int) {
	p.getUENBSrv().HistorySetSize(size)
}

// NBState 推送基站状态
func (p *Push) NBState(nbState *model.NBState) error {
	var baseURL, neUid string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
		neUid = c.OMC.NeUID
	})

	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}

	if nbState.NeUid == "" {
		nbState.NeUid = neUid
	}

	url := p.joinURL(baseURL, service.NB_STATE_PUSH_URI)
	return p.getNBStateSrv().PushURL(url, nbState, 0)
}

// NBStateURL 推送基站状态自定义URL
func (p *Push) NBStateURL(url string, nbState *model.NBState, timeout time.Duration) error {
	var neUid string
	p.o.cfg.View(func(c *config.Config) {
		neUid = c.OMC.NeUID
	})
	if nbState.NeUid == "" {
		nbState.NeUid = neUid
	}
	return p.getNBStateSrv().PushURL(url, nbState, timeout)
}

// NBStateHistoryList 获取状态推送历史
func (p *Push) NBStateHistoryList(n int) []model.NBState {
	return p.getNBStateSrv().HistoryList(n)
}

// NBStateHistorySetSize 设置状态推送历史大小
func (p *Push) NBStateHistorySetSize(size int) {
	p.getNBStateSrv().HistorySetSize(size)
}

// CDR 推送话单
func (p *Push) CDR(cdr *model.CDR) error {
	var baseURL, neUid string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
		neUid = c.OMC.NeUID
	})

	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}

	if cdr.NeUid == "" {
		cdr.NeUid = neUid
	}

	url := p.joinURL(baseURL, service.CDR_PUSH_URI)
	return p.getCDRSrv().PushURL(url, cdr, 0)
}

// CDRURL 推送话单自定义URL
func (p *Push) CDRURL(url string, cdr *model.CDR, timeout time.Duration) error {
	var neUid string
	p.o.cfg.View(func(c *config.Config) {
		neUid = c.OMC.NeUID
	})
	if cdr.NeUid == "" {
		cdr.NeUid = neUid
	}
	return p.getCDRSrv().PushURL(url, cdr, timeout)
}

// CDRHistoryList 获取话单推送历史
func (p *Push) CDRHistoryList(n int) []model.CDR {
	return p.getCDRSrv().HistoryList(n)
}

// CDRHistorySetSize 设置话单推送历史大小
func (p *Push) CDRHistorySetSize(size int) {
	p.getCDRSrv().HistorySetSize(size)
}

// KPI 推送指标
func (p *Push) KPI() error {
	var baseURL string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
	})

	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}

	url := p.joinURL(baseURL, service.KPI_PUSH_URI)
	return p.getKPISrv().PushURL(url, 0)
}

// KPIURL 推送指标自定义URL
func (p *Push) KPIURL(url string, timeout time.Duration) error {
	return p.getKPISrv().PushURL(url, timeout)
}

// KPIHistoryList 获取 KPI 推送历史
func (p *Push) KPIHistoryList(n int) []model.KPI {
	return p.getKPISrv().HistoryList(n)
}

// KPIHistorySetSize 设置 KPI 推送历史大小
func (p *Push) KPIHistorySetSize(size int) {
	p.getKPISrv().HistorySetSize(size)
}

// KPITimerStart 开启 KPI 定时推送
func (p *Push) KPITimerStart() error {
	var baseURL string
	p.o.cfg.View(func(c *config.Config) {
		baseURL = c.OMC.URL
	})
	if baseURL == "" {
		return fmt.Errorf("OMC URL is empty")
	}
	url := p.joinURL(baseURL, service.KPI_PUSH_URI)
	p.getKPISrv().KPITimerStart(url)
	return nil
}

// KPITimerStop 停止 KPI 定时推送
func (p *Push) KPITimerStop() {
	p.getKPISrv().KPITimerStop()
}

// KPIKeySet 设置 KPI 键值
func (p *Push) KPIKeySet(key string, v float64) {
	p.getKPISrv().KeySet(key, v)
}

// KPIKeyGet 获取 KPI 键值
func (p *Push) KPIKeyGet(key string) float64 {
	return p.getKPISrv().KeyGet(key)
}

// KPIKeyInc KPI 键值自增 1
func (p *Push) KPIKeyInc(key string) {
	p.getKPISrv().KeyInc(key)
}

// KPIKeyDec KPI 键值自减 1
func (p *Push) KPIKeyDec(key string) {
	p.getKPISrv().KeyDec(key)
}

// KPIKeyAdd KPI 键值增加
func (p *Push) KPIKeyAdd(key string, v float64) {
	p.getKPISrv().KeyAdd(key, v)
}

// KPIKeyDel 删除 KPI 键
func (p *Push) KPIKeyDel(key string) {
	p.getKPISrv().KeyDel(key)
}
