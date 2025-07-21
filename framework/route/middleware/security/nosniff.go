package security

import (
	"github.com/tsmask/go-oam/framework/config"

	"github.com/gin-gonic/gin"
)

// nosniff 用于防止 XSS 等跨站脚本攻击
// 如果从 script 或 stylesheet 读入的文件的 MIME 类型与指定 MIME 类型不匹配，不允许读取该文件。
func nosniff(c *gin.Context) {
	// 排除状态码范围
	status := c.Writer.Status()
	if status >= 300 && status <= 308 {
		return
	}

	enable := false
	if v := config.Get("security.nosniff.enable"); v != nil {
		enable = v.(bool)
	}

	if enable {
		c.Header("x-content-type-options", "nosniff")
	}
}
