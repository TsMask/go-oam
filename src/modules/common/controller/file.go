package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tsmask/go-oam/src/framework/route/resp"
	"github.com/tsmask/go-oam/src/framework/utils/file"

	"github.com/gin-gonic/gin"
)

// 实例化控制层 FileController 结构体
var NewFile = &FileController{}

// 文件操作处理
//
// PATH /
type FileController struct{}

// 上传文件
//
// POST /upload
//
//	@Tags			common/file
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"The file to upload."
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Upload a file
//	@Description	Upload a file, interface param use <fileName>
//	@Router			/file/upload [post]
func (s *FileController) Upload(c *gin.Context) {
	// 上传的文件
	formFile, err := c.FormFile("file")
	if err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 上传文件转存
	uploadFilePath, err := file.TransferUploadFile(formFile, []string{})
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}

	c.JSON(200, resp.OkData(map[string]string{
		"url":              "//" + c.Request.Host + uploadFilePath,
		"filePath":         uploadFilePath,
		"newFileName":      filepath.Base(uploadFilePath),
		"originalFileName": formFile.Filename,
	}))
}

// 切片文件检查
//
// POST /chunk-check
//
//	@Tags			common/file
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Slice file checking
//	@Description	Slice file checking
//	@Router			/file/chunk-check [post]
func (s *FileController) ChunkCheck(c *gin.Context) {
	var body struct {
		Identifier string `json:"identifier" binding:"required"` // 唯一标识
		FileName   string `json:"fileName" binding:"required"`   // 文件名
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 读取标识目录
	chunks, err := file.ChunkCheckFile(body.Identifier, body.FileName)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.OkData(chunks))
}

// 切片文件合并
//
// POST /chunk-merge
//
//	@Tags			common/file
//	@Accept			json
//	@Produce		json
//	@Param			data	body		object	true	"Request Param"
//	@Success		200		{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Slice file merge
//	@Description	Slice file merge
//	@Router			/file/chunk-merge [post]
func (s *FileController) ChunkMerge(c *gin.Context) {
	var body struct {
		Identifier string `json:"identifier" binding:"required"` // 唯一标识
		FileName   string `json:"fileName" binding:"required"`   // 文件名
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 切片文件合并
	mergeFilePath, err := file.ChunkMergeFile(body.Identifier, body.FileName)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}

	c.JSON(200, resp.OkData(map[string]string{
		"url":              "//" + c.Request.Host + mergeFilePath,
		"filePath":         mergeFilePath,
		"newFileName":      filepath.Base(mergeFilePath),
		"originalFileName": body.FileName,
	}))
}

// 切片文件上传
//
// POST /chunk-upload
//
//	@Tags			common/file
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file		formData	file	true	"The file to upload."
//	@Param			identifier	formData	string	true	"Slice Marker"
//	@Param			index		formData	string	true	"Slice No."
//	@Success		200			{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Sliced file upload
//	@Description	Sliced file upload
//	@Router			/file/chunk-upload [post]
func (s *FileController) ChunkUpload(c *gin.Context) {
	// 切片编号
	index := c.PostForm("index")
	// 切片唯一标识
	identifier := c.PostForm("identifier")
	if index == "" || identifier == "" {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, "bind err: index and identifier must be set"))
		return
	}
	// 上传的文件
	formFile, err := c.FormFile("file")
	if err != nil {
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_CHEACK, "bind err: file is empty"))
		return
	}

	// 上传文件转存
	chunkFilePath, err := file.TransferChunkUploadFile(formFile, index, identifier)
	if err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(206, resp.OkData(chunkFilePath))
}

// 本地文件列表
//
// GET /list
//
//	@Tags			common/file
//	@Accept			json
//	@Produce		json
//	@Param			path		query		string	true	"file path"		default(/var/log)
//	@Param			pageNum		query		number	true	"pageNum"		default(1)
//	@Param			pageSize	query		number	true	"pageSize"		default(10)
//	@Param			search		query		string	false	"search prefix"	default(upf)
//	@Success		200			{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Local file list
//	@Description	Local file list
//	@Router			/file/list [get]
func (s *FileController) List(c *gin.Context) {
	var querys struct {
		Path     string `form:"path" binding:"required"` // 路径
		PageNum  int64  `form:"pageNum" binding:"required"`
		PageSize int64  `form:"pageSize" binding:"required"`
		Search   string `form:"search"` // 文件名前缀匹配
	}
	if err := c.ShouldBindQuery(&querys); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 获取文件列表
	localFilePath := querys.Path
	if runtime.GOOS == "windows" {
		localFilePath = fmt.Sprintf("C:%s", localFilePath)
	}
	rows, err := file.FileList(localFilePath, querys.Search)
	if err != nil {
		c.JSON(200, resp.OkData(map[string]any{
			"path":  querys.Path,
			"total": len(rows),
			"rows":  []file.FileListRow{},
		}))
		return
	}

	// 对数组进行切片分页
	lenNum := int64(len(rows))
	start := (querys.PageNum - 1) * querys.PageSize
	end := start + querys.PageSize
	var splitRows []file.FileListRow
	if start >= lenNum {
		splitRows = []file.FileListRow{}
	} else if end >= lenNum {
		splitRows = rows[start:]
	} else {
		splitRows = rows[start:end]
	}

	c.JSON(200, resp.OkData(map[string]any{
		"path":  querys.Path,
		"total": lenNum,
		"rows":  splitRows,
	}))
}

// 本地文件获取下载
//
// GET /
//
//	@Tags			common/file
//	@Accept			json
//	@Produce		json
//	@Param			path		query		string	true	"file path"		default(/var/log)
//	@Param			fileName		query		string	true	"file name"		default(oam.log)
//	@Success		200			{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Local files for download
//	@Description	Local files for download
//	@Router			/file [get]
func (s *FileController) File(c *gin.Context) {
	var querys struct {
		Path     string `form:"path"  binding:"required"`     // 路径
		Filename string `form:"fileName"  binding:"required"` // 文件名
	}
	if err := c.ShouldBindQuery(&querys); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 检查路径是否在允许的目录范围内
	// allowedPaths := []string{"/var/log", "/tmp"}
	// allowed := false
	// for _, p := range allowedPaths {
	// 	if strings.HasPrefix(querys.Path, p) {
	// 		allowed = true
	// 		break
	// 	}
	// }
	// if !allowed {
	// 	c.JSON(200, resp.ErrMsg("operation path is not within the allowed range"))
	// 	return
	// }

	// 获取文件路径并下载
	localFilePath := filepath.Join(querys.Path, querys.Filename)
	if runtime.GOOS == "windows" {
		localFilePath = fmt.Sprintf("C:%s", localFilePath)
	}
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		c.JSON(200, resp.ErrMsg("file does not exist"))
		return
	}
	c.FileAttachment(localFilePath, querys.Filename)
}

// 本地文件删除
//
// DELETE /
//
//	@Tags			common/file
//	@Accept			json
//	@Produce		json
//	@Param			path		query		string	true	"file path"		default(/var/log)
//	@Param			fileName		query		string	true	"file name"		default(oam.log)
//	@Success		200			{object}	object	"Response Results"
//	@Security		TokenAuth
//	@Summary		Local file deletion
//	@Description	Local file deletion
//	@Router			/file [delete]
func (s *FileController) Remove(c *gin.Context) {
	var querys struct {
		Path     string `form:"path"  binding:"required"`
		Filename string `form:"fileName"  binding:"required"`
	}
	if err := c.ShouldBindQuery(&querys); err != nil {
		errMsgs := fmt.Sprintf("bind err: %s", resp.FormatBindError(err))
		c.JSON(422, resp.CodeMsg(resp.CODE_PARAM_PARSER, errMsgs))
		return
	}

	// 检查路径是否在允许的目录范围内
	// allowedPaths := []string{"/tmp"}
	// allowed := false
	// for _, p := range allowedPaths {
	// 	if strings.HasPrefix(querys.Path, p) {
	// 		allowed = true
	// 		break
	// 	}
	// }
	// if !allowed {
	// 	c.JSON(200, resp.ErrMsg("operation path is not within the allowed range"))
	// 	return
	// }

	// 获取文件路径并删除
	localFilePath := filepath.Join(querys.Path, querys.Filename)
	if runtime.GOOS == "windows" {
		localFilePath = fmt.Sprintf("C:%s", localFilePath)
	}
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		c.JSON(200, resp.ErrMsg("file does not exist"))
		return
	}
	if err := os.Remove(localFilePath); err != nil {
		c.JSON(200, resp.ErrMsg(err.Error()))
		return
	}
	c.JSON(200, resp.Ok(nil))
}
