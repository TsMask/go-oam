package file

import (
	"fmt"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/utils/generate"
	"github.com/tsmask/go-oam/framework/utils/parse"
)

/**最大文件名长度 */
const DEFAULT_FILE_NAME_LENGTH = 100

// UploadFileListRow 文件列表行
type UploadFileListRow struct {
	Name    string `json:"name"`    // 文件名
	Size    int64  `json:"size"`    // 文件大小
	ModTime int64  `json:"modTime"` // 修改时间
	IsDir   bool   `json:"isDir"`   // 是否目录
}

// 最大上传文件大小
func uploadFileSize(cfg *config.Config) int64 {
	var size int64
	cfg.View(func(c *config.Config) {
		size = int64(c.Upload.FileSize)
	})
	if size < 1 {
		size = 1
	}
	return size * 1024 * 1024
}

// 上传文件资源路径
func uploadFileDir(cfg *config.Config) string {
	var fileDir string
	cfg.View(func(c *config.Config) {
		fileDir = c.Upload.FileDir
	})
	if fileDir == "" {
		fileDir = "/tmp"
	}
	return fileDir
}

// 文件上传扩展名白名单
func uploadWhiteList(cfg *config.Config) []string {
	var list []string
	cfg.View(func(c *config.Config) {
		list = c.Upload.WhiteList
	})
	return list
}

// 生成文件名称 fileName_随机值.extName
//
// fileName 原始文件名称含后缀，如：logo.png
func generateFileName(fileName string) string {
	fileExt := filepath.Ext(fileName)
	// 去除后缀
	regex := regexp.MustCompile(fileExt)
	newFileName := regex.ReplaceAllString(fileName, "")
	// 去除非法字符
	regex = regexp.MustCompile(`[\\/:*?"<>|]`)
	newFileName = regex.ReplaceAllString(newFileName, "")
	// 去除空格
	regex = regexp.MustCompile(`\s`)
	newFileName = regex.ReplaceAllString(newFileName, "_")
	newFileName = strings.TrimSpace(newFileName)
	return fmt.Sprintf("%s_%s%s", newFileName, generate.Code(6), fileExt)
}

// 检查文件允许写入本地
//
// fileName 原始文件名称含后缀，如：oam_logo_iipc68.png
//
// allowExts 允许上传拓展类型，['.png']
func IsAllowWrite(cfg *config.Config, fileName string, allowExts []string, fileSize int64) error {
	// 判断上传文件名称长度
	if len(fileName) > DEFAULT_FILE_NAME_LENGTH {
		return fmt.Errorf("the maximum length limit for uploading file names is %d", DEFAULT_FILE_NAME_LENGTH)
	}

	// 最大上传文件大小
	maxFileSize := uploadFileSize(cfg)
	if fileSize > maxFileSize {
		return fmt.Errorf("maximum upload file size %s", parse.Bit(float64(maxFileSize)))
	}

	// 判断文件拓展是否为允许的拓展类型
	fileExt := filepath.Ext(fileName)
	hasExt := false
	if len(allowExts) == 0 {
		allowExts = uploadWhiteList(cfg)
	}
	if slices.Contains(allowExts, fileExt) {
		hasExt = true
	}

	if !hasExt {
		return fmt.Errorf("unsupported upload file extensions %s", fileExt)
	}

	return nil
}

// GetFilePath 获取文件存储路径
func GetFilePath(cfg *config.Config, fileName string) string {
	dir := uploadFileDir(cfg)
	// 如果目录不存在，尝试创建
	// os.MkdirAll(dir, 0755)
	return path.Join(dir, fileName)
}

// Save 上传文件保存到本地
func Save(cfg *config.Config, file *multipart.FileHeader) (string, error) {
	// 校验文件是否允许上传
	if err := IsAllowWrite(cfg, file.Filename, []string{}, file.Size); err != nil {
		return "", err
	}
	// 生成文件名称
	fileName := generateFileName(file.Filename)
	// 文件存储路径
	filePath := GetFilePath(cfg, fileName)
	return filePath, nil
}

// TransferUploadFile 上传文件转存
func TransferUploadFile(cfg *config.Config, file *multipart.FileHeader, allowExts []string) (string, error) {
	if err := IsAllowWrite(cfg, file.Filename, allowExts, file.Size); err != nil {
		return "", err
	}
	fileName := generateFileName(file.Filename)
	filePath := GetFilePath(cfg, fileName)
	if err := transferToNewFile(file, filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

// TransferUploadBytes 上传字节转存
func TransferUploadBytes(cfg *config.Config, fileName string, bin []byte, allowExts []string) (string, error) {
	if err := IsAllowWrite(cfg, fileName, allowExts, int64(len(bin))); err != nil {
		return "", err
	}
	newFileName := generateFileName(fileName)
	filePath := GetFilePath(cfg, newFileName)
	if err := writeBytesToFile(filePath, bin); err != nil {
		return "", err
	}
	return filePath, nil
}

// TransferChunkUploadFile 切片文件上传转存
func TransferChunkUploadFile(cfg *config.Config, file *multipart.FileHeader, index string, identifier string) (string, error) {
	// 校验分片大小
	if file.Size > 10*1024*1024 {
		return "", fmt.Errorf("chunk size exceeds 10MB limit")
	}

	dir := uploadFileDir(cfg)
	chunkDir := filepath.Join(dir, identifier)
	chunkPath := filepath.Join(chunkDir, index)

	if err := transferToNewFile(file, chunkPath); err != nil {
		return "", err
	}
	return chunkPath, nil
}

// TransferChunkUploadBytes 切片字节上传转存
func TransferChunkUploadBytes(cfg *config.Config, fileName string, index string, identifier string, bin []byte) (string, error) {
	// 校验分片大小
	if len(bin) > 2*1024*1024 {
		return "", fmt.Errorf("chunk size exceeds 2MB limit")
	}

	dir := uploadFileDir(cfg)
	chunkDir := filepath.Join(dir, identifier)
	chunkPath := filepath.Join(chunkDir, index)

	if err := writeBytesToFile(chunkPath, bin); err != nil {
		return "", err
	}
	return chunkPath, nil
}

// ChunkCheckFile 切片文件检查
func ChunkCheckFile(cfg *config.Config, identifier string, fileName string) ([]string, error) {
	dir := uploadFileDir(cfg)
	chunkDir := filepath.Join(dir, identifier)
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		return []string{}, nil
	}
	return getDirFileNameList(chunkDir)
}

// ChunkMergeFile 切片文件合并
func ChunkMergeFile(cfg *config.Config, identifier string, fileName string) (string, error) {
	dir := uploadFileDir(cfg)
	chunkDir := filepath.Join(dir, identifier)
	newFileName := generateFileName(fileName)
	mergeFilePath := GetFilePath(cfg, newFileName)

	if err := mergeToNewFile(chunkDir, mergeFilePath); err != nil {
		return "", err
	}
	return mergeFilePath, nil
}

// ReadUploadFileStream 读取上传文件流
func ReadUploadFileStream(cfg *config.Config, filePath string, rangeStr string) (map[string]any, error) {
	// 简单解析 Range: bytes=start-end
	var start, end int64
	if rangeStr != "" {
		fmt.Sscanf(rangeStr, "bytes=%d-%d", &start, &end)
	}

	data, err := getFileStream(filePath, start, end)
	if err != nil {
		return nil, err
	}

	fileSize := getFileSize(filePath)
	if end == 0 || end >= fileSize {
		end = fileSize - 1
	}

	return map[string]any{
		"data":  data,
		"start": start,
		"end":   end,
		"total": fileSize,
	}, nil
}

// UploadFileList 获取本地文件列表
func UploadFileList(dirPath string, search string) ([]UploadFileListRow, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var rows []UploadFileListRow
	for _, f := range files {
		if search != "" && !strings.HasPrefix(f.Name(), search) {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		rows = append(rows, UploadFileListRow{
			Name:    f.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			IsDir:   f.IsDir(),
		})
	}
	return rows, nil
}
