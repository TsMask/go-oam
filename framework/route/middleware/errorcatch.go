package middleware

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// ErrorCatch 全局异常捕获
func ErrorCatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// 在这里处理 Panic 异常，例如记录日志或返回错误信息给客户端
			if err := recover(); err != nil {
				fmt.Printf("[OAM] Panic Error %s %s \n %v\n", c.Request.Method, c.Request.URL, err)
				c.JSON(500, resp.CodeMsg(resp.CODE_INTERNAL, resp.MSG_INTERNAL))
				c.Abort() // 停止执行后续的处理函数
			}
		}()

		c.Next()
	}
}
