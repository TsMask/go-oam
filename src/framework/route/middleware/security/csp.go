package security

import (
	"fmt"
	"time"

	"github.com/tsmask/go-oam/src/framework/config"

	"github.com/gin-gonic/gin"
)

// TODO
// csp 这将帮助防止跨站脚本攻击（XSS）。
// HTTP 响应头 Content-Security-Policy 允许站点管理者控制指定的页面加载哪些资源。
func csp(c *gin.Context) {
	enable := false
	if v := config.Get("security.csp.enable"); v != nil {
		enable = v.(bool)
	}

	if enable {
		c.Header("x-csp-nonce", fmt.Sprintf("%d", time.Now().UnixNano()%100000000))
	}
}
