package common

import (
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/modules/common/controller"

	"github.com/gin-gonic/gin"
)

// 模块路由注册
func SetupRoute(router gin.IRouter) {
	logger.Infof("开始加载 ====> common 模块路由")

	// 路由主页
	router.GET("/i", controller.NewIndex.Handler)

	// 路由服务器时间
	router.GET("/time", controller.NewTimestamp.Handler)

	// 文件操作处理
	file := controller.NewFile
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
