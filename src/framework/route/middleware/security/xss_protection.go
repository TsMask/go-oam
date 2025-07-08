package security

import (
	"github.com/tsmask/go-oam/src/framework/config"

	"github.com/gin-gonic/gin"
)

// xssProtection 用于启用浏览器的XSS过滤功能，以防止 XSS 跨站脚本攻击。
func xssProtection(c *gin.Context) {
	enable := false
	if v := config.Get("security.xssProtection.enable"); v != nil {
		enable = v.(bool)
	}

	value := "1; mode=block"
	if v := config.Get("security.xssProtection.value"); v != nil {
		value = v.(string)
	}

	if enable {
		c.Header("x-xss-protection", value)
	}
}
