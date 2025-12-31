package common

import (
	"github.com/tsmask/go-oam/modules/common/controller"

	"github.com/gin-gonic/gin"
)

// SetupRoute 模块路由注册
func SetupRoute(router gin.IRouter) {
	index := controller.NewIndexController()
	timestamp := controller.NewTimestampController()
	file := controller.NewFileController()

	// 路由主页
	router.GET("/i", index.Handler)

	// 路由服务器时间
	router.GET("/time", timestamp.Handler)

	// 文件操作处理
	fileGroup := router.Group("/file")
	{
		fileGroup.POST("/upload", file.Upload)
		fileGroup.POST("/chunk-check", file.ChunkCheck)
		fileGroup.POST("/chunk-upload", file.ChunkUpload)
		fileGroup.POST("/chunk-merge", file.ChunkMerge)
		fileGroup.GET("/list", file.List)
		fileGroup.GET("", file.File)
		fileGroup.DELETE("", file.Remove)
	}
}
