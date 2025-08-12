package security

import (
	"github.com/gin-gonic/gin"
)

// noopen 用于指定 IE 8 以上版本的用户不打开文件而直接保存文件。
// 在下载对话框中不显式“打开”选项。
func noopen(c *gin.Context, opt NoOpen) {
	if !opt.Enable {
		return
	}

	c.Header("x-download-options", "noopen")
}
