package security

import (
	"github.com/gin-gonic/gin"
)

// xframe 用来配置 X-Frame-Options 响应头
// 用来给浏览器指示允许一个页面可否在 frame, iframe, embed 或者 object 中展现的标记。
// 站点可以通过确保网站没有被嵌入到别人的站点里面，从而避免 clickjacking 攻击。
func xframe(c *gin.Context, opt XFrame) {
	if !opt.Enable {
		return
	}

	value := "sameorigin"
	if opt.Value != "" {
		value = opt.Value
	}
	c.Header("x-frame-options", value)
}
