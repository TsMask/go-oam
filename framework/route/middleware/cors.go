package middleware

import (
	"fmt"
	"strings"

	"github.com/tsmask/go-oam/framework/config"

	"github.com/gin-gonic/gin"
)

// Cors 跨域
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置Vary头部
		c.Header("Vary", "Origin")
		c.Header("Keep-Alive", "timeout=5")

		requestOrigin := c.GetHeader("Origin")
		if requestOrigin == "" {
			c.Next()
			return
		}

		origin := requestOrigin
		if v := config.Get("cors.origin"); v != nil {
			origin = v.(string)
		}
		c.Header("Access-Control-Allow-Origin", origin)

		if v := config.Get("cors.credentials"); v != nil && v.(bool) {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// OPTIONS
		if method := c.Request.Method; method == "OPTIONS" {
			requestMethod := c.GetHeader("Access-Control-Request-Method")
			if requestMethod == "" {
				c.Next()
				return
			}

			// 响应最大时间值
			if v := config.Get("cors.maxAge"); v != nil && v.(int) > 10000 {
				c.Header("Access-Control-Max-Age", fmt.Sprint(v))
			}

			// 允许方法
			if v := config.Get("cors.allowMethods"); v != nil {
				var allowMethods = make([]string, 0)
				for _, s := range v.([]any) {
					allowMethods = append(allowMethods, s.(string))
				}
				c.Header("Access-Control-Allow-Methods", strings.Join(allowMethods, ","))
			} else {
				c.Header("Access-Control-Allow-Methods", "GET,HEAD,PUT,POST,DELETE,PATCH")
			}

			// 允许请求头
			if v := config.Get("cors.allowHeaders"); v != nil {
				var allowHeaders = make([]string, 0)
				for _, s := range v.([]any) {
					allowHeaders = append(allowHeaders, s.(string))
				}
				c.Header("Access-Control-Allow-Headers", strings.Join(allowHeaders, ","))
			}

			c.AbortWithStatus(204)
			return
		}

		// 暴露请求头
		if v := config.Get("cors.exposeHeaders"); v != nil {
			var exposeHeaders = make([]string, 0)
			for _, s := range v.([]any) {
				exposeHeaders = append(exposeHeaders, s.(string))
			}
			c.Header("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ","))
		}

		c.Next()
	}
}
