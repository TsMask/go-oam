package security

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// TODO
// csp 这将帮助防止跨站脚本攻击（XSS）。
// HTTP 响应头 Content-Security-Policy 允许站点管理者控制指定的页面加载哪些资源。
func csp(c *gin.Context, opt CSP) {
	if !opt.Enable {
		return
	}

	c.Header("x-csp-nonce", fmt.Sprintf("%d", time.Now().UnixMilli()))
}
