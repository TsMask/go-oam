package security

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// hsts 是一个安全功能 HTTP Strict Transport Security（通常简称为 HSTS ）
// 它告诉浏览器只能通过 HTTPS 访问当前资源，而不是 HTTP。
func hsts(c *gin.Context, opt HSTS) {
	if !opt.Enable {
		return
	}

	maxAge := 365 * 24 * 3600
	if v := opt.MaxAge; v > 1000 {
		maxAge = v
	}

	str := fmt.Sprintf("max-age=%d", maxAge)
	if opt.IncludeSubdomains {
		str += "; includeSubdomains"
	}
	c.Header("strict-transport-security", str)
}
