package security

import (
	"github.com/tsmask/go-oam/src/framework/config"

	"github.com/gin-gonic/gin"
)

// noopen 用于指定 IE 8 以上版本的用户不打开文件而直接保存文件。
// 在下载对话框中不显式“打开”选项。
func noopen(c *gin.Context) {
	enable := false
	if v := config.Get("security.noopen.enable"); v != nil {
		enable = v.(bool)
	}

	if enable {
		c.Header("x-download-options", "noopen")
	}
}
