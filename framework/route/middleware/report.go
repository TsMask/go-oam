package middleware

import (
	"log"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// Report 请求响应日志
func Report(tag string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 调用下一个处理程序
		c.Next()

		// 计算请求处理时间，并打印日志
		duration := time.Since(start)
		numGoroutines := runtime.NumGoroutine()
		log.Printf("[%s] Report\nAPI: %s %s\nTotal Duration: %v\nCurrently Active Goroutines: %d\n", tag, c.Request.Method, c.Request.RequestURI, duration, numGoroutines)
	}
}
