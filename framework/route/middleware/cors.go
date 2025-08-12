package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// CorsOpt 跨域配置
type CorsOpt struct {
	// 设置 Access-Control-Allow-Origin 的值 例如：http://go-oam.org
	// 如果请求设置了 credentials，则 origin 不能设置为 *
	Origin string
	// 设置 Access-Control-Allow-Credentials
	Credentials bool
	// 设置 Access-Control-Max-Age
	MaxAge int
	// 允许跨域的方法
	AllowMethods []string
	// 设置 Access-Control-Allow-Headers 的值
	AllowHeaders []string
	// 设置 Access-Control-Expose-Headers 的值
	ExposeHeaders []string
}

// CorsDefaultOpt 默认跨域配置
var CorsDefaultOpt = CorsOpt{
	Origin:       "*",
	Credentials:  true,
	MaxAge:       31536000,
	AllowMethods: []string{"OPTIONS", "GET", "HEAD", "PUT", "POST", "DELETE", "PATCH"},
	AllowHeaders: []string{
		"X-App-Code",
		"X-App-Version",
		"X-Requested-With",
		"Authorization",
		"Origin",
		"Content-Type",
		"Content-Language",
		"Accept-Language",
		"Accept",
		"Range",
	},
	ExposeHeaders: []string{
		"X-RepeatSubmit-Rest",
	},
}

// Cors 跨域
func Cors(opt CorsOpt) gin.HandlerFunc {
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
		if v := opt.Origin; v != "" {
			origin = v
		}
		c.Header("Access-Control-Allow-Origin", origin)

		if opt.Credentials {
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
			if v := opt.MaxAge; v > 10000 {
				c.Header("Access-Control-Max-Age", fmt.Sprint(v))
			}

			// 允许方法
			if v := opt.AllowMethods; len(v) > 0 {
				var allowMethods = make([]string, 0)
				allowMethods = append(allowMethods, v...)
				c.Header("Access-Control-Allow-Methods", strings.Join(allowMethods, ","))
			} else {
				c.Header("Access-Control-Allow-Methods", "GET,HEAD,PUT,POST,DELETE,PATCH")
			}

			// 允许请求头
			if v := opt.AllowHeaders; len(v) > 0 {
				var allowHeaders = make([]string, 0)
				allowHeaders = append(allowHeaders, v...)
				c.Header("Access-Control-Allow-Headers", strings.Join(allowHeaders, ","))
			}

			c.AbortWithStatus(204)
			return
		}

		// 暴露请求头
		if v := opt.ExposeHeaders; len(v) > 0 {
			var exposeHeaders = make([]string, 0)
			exposeHeaders = append(exposeHeaders, v...)
			c.Header("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ","))
		}

		c.Next()
	}
}
