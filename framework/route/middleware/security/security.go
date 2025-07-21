package security

import (
	"github.com/gin-gonic/gin"
)

// Security 安全
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 拦截，判断是否有效Referer
		referer(c)

		// 无拦截，仅仅设置响应头
		xframe(c)
		csp(c)
		hsts(c)
		noopen(c)
		nosniff(c)
		xssProtection(c)

		c.Next()
	}
}
