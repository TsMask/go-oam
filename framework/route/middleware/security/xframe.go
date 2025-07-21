package security

import (
	"github.com/tsmask/go-oam/framework/config"

	"github.com/gin-gonic/gin"
)

// xframe 用来配置 X-Frame-Options 响应头
// 用来给浏览器指示允许一个页面可否在 frame, iframe, embed 或者 object 中展现的标记。
// 站点可以通过确保网站没有被嵌入到别人的站点里面，从而避免 clickjacking 攻击。
func xframe(c *gin.Context) {
	enable := false
	if v := config.Get("security.xframe.enable"); v != nil {
		enable = v.(bool)
	}

	value := "sameorigin"
	if v := config.Get("security.xframe.value"); v != nil {
		value = v.(string)
	}

	if enable {
		c.Header("x-frame-options", value)
	}
}
