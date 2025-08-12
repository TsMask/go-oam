package middleware

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// Report 请求响应日志
func Report() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 调用下一个处理程序
		c.Next()

		// 计算请求处理时间，并打印日志
		duration := time.Since(start)
		numGoroutines := runtime.NumGoroutine()
		fmt.Printf("\n访问接口: %s %s\n总耗时: %v\n当前活跃的Goroutine数量: %d", c.Request.Method, c.Request.RequestURI, duration, numGoroutines)
	}
}
