package security

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/config"

	"github.com/gin-gonic/gin"
)

// hsts 是一个安全功能 HTTP Strict Transport Security（通常简称为 HSTS ）
// 它告诉浏览器只能通过 HTTPS 访问当前资源，而不是 HTTP。
func hsts(c *gin.Context) {
	enable := false
	if v := config.Get("security.hsts.enable"); v != nil {
		enable = v.(bool)
	}

	maxAge := 365 * 24 * 3600
	if v := config.Get("security.hsts.maxAge"); v != nil {
		maxAge = v.(int)
	}

	includeSubdomains := false
	if v := config.Get("security.hsts.includeSubdomains"); v != nil {
		includeSubdomains = v.(bool)
	}

	str := fmt.Sprintf("max-age=%d", maxAge)
	if includeSubdomains {
		str += "; includeSubdomains"
	}

	if enable {
		c.Header("strict-transport-security", str)
	}
}
