package security

import (
	"net/url"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// referer 配置 referer 的 host 部分
func referer(c *gin.Context) {
	enable := false
	if v := config.Get("security.csrf.enable"); v != nil {
		enable = v.(bool)
	}
	if !enable {
		return
	}

	// csrf 校验类型
	okType := false
	if v := config.Get("security.csrf.type"); v != nil {
		vType := v.(string)
		if vType == "all" || vType == "any" || vType == "referer" {
			okType = true
		}
	}
	if !okType {
		return
	}

	// 忽略请求方法
	method := c.Request.Method
	ignoreMethods := []string{"GET", "HEAD", "OPTIONS", "TRACE"}
	for _, ignore := range ignoreMethods {
		if ignore == method {
			return
		}
	}

	referer := c.GetHeader("Referer")
	if referer == "" {
		// 无效 Referer 未知
		c.AbortWithStatusJSON(200, resp.ErrMsg("Invalid referer unknown"))
		return
	}

	// 获取host
	u, err := url.Parse(referer)
	if err != nil {
		// 无效 Referer 未知
		c.AbortWithStatusJSON(200, resp.ErrMsg("Invalid referer unknown"))
		return
	}
	host := u.Host

	// 允许的来源白名单
	refererWhiteList := make([]string, 0)
	if v := config.Get("security.csrf.refererWhiteList"); v != nil {
		for _, s := range v.([]any) {
			refererWhiteList = append(refererWhiteList, s.(string))
		}
	}

	// 遍历检查
	ok := false
	for _, domain := range refererWhiteList {
		if domain == host {
			ok = true
		}
	}
	if !ok {
		// 无效 Referer
		c.AbortWithStatusJSON(200, resp.ErrMsg("Invalid referer "+host))
		return
	}
}
