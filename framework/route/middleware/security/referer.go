package security

import (
	"net/url"

	"github.com/tsmask/go-oam/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// referer 配置 referer 的 host 部分
func referer(c *gin.Context, opt CSRF) {
	if !opt.Enable {
		return
	}

	// csrf 校验类型
	okType := false
	if v := opt.Type; v != "" {
		if v == "all" || v == "any" || v == "referer" {
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
		c.AbortWithStatusJSON(200, resp.ErrMsg("invalid referer unknown"))
		return
	}

	// 获取host
	u, err := url.Parse(referer)
	if err != nil {
		// 无效 Referer 未知
		c.AbortWithStatusJSON(200, resp.ErrMsg("invalid referer unknown"))
		return
	}
	host := u.Host

	// 允许的来源白名单
	refererWhiteList := make([]string, 0)
	refererWhiteList = append(refererWhiteList, opt.RefererWhiteList...)

	// 遍历检查
	ok := false
	for _, domain := range refererWhiteList {
		if domain == host {
			ok = true
		}
	}
	if !ok {
		// 无效 Referer
		c.AbortWithStatusJSON(200, resp.ErrMsg("invalid referer "+host))
		return
	}
}
