package middleware

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/logger"
	"github.com/tsmask/go-oam/framework/route/resp"

	"github.com/gin-gonic/gin"
)

// ErrorCatch 全局异常捕获
func ErrorCatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// 在这里处理 Panic 异常，例如记录日志或返回错误信息给客户端
			if err := recover(); err != nil {
				logger.Errorf("Panic Error %s %s => %v", c.Request.Method, c.Request.URL, err)

				// 返回错误响应给客户端
				if config.Dev() {
					// 通过实现 error 接口的 Error() 方法自定义错误类型进行捕获
					switch v := err.(type) {
					case error:
						c.JSON(500, resp.CodeMsg(resp.CODE_INTERNAL, v.Error()))
					default:
						c.JSON(500, resp.CodeMsg(resp.CODE_INTERNAL, fmt.Sprint(err)))
					}
				} else {
					c.JSON(500, resp.CodeMsg(resp.CODE_INTERNAL, resp.MSG_INTERNAL))
				}

				c.Abort() // 停止执行后续的处理函数
			}
		}()

		c.Next()
	}
}
