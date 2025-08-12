package security

import (
	"github.com/gin-gonic/gin"
)

// xssProtection 用于启用浏览器的XSS过滤功能，以防止 XSS 跨站脚本攻击。
func xssProtection(c *gin.Context, opt XSSProtection) {
	if !opt.Enable {
		return
	}

	value := "1; mode=block"
	if v := opt.Value; v != "" {
		value = v
	}
	c.Header("x-xss-protection", value)
}
