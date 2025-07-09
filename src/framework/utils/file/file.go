package file

import (
	"fmt"
	"mime/multipart"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tsmask/go-oam/src/framework/config"
	"github.com/tsmask/go-oam/src/framework/utils/date"
	"github.com/tsmask/go-oam/src/framework/utils/generate"
	"github.com/tsmask/go-oam/src/framework/utils/parse"
)

/**最大文件名长度 */
const DEFAULT_FILE_NAME_LENGTH = 100

// 最大上传文件大小
func uploadFileSize() int64 {
	fileSize := 1 * 1024 * 1024
	size := config.Get("upload.fileSize").(int)
	if size > 1 {
		fileSize = size * 1024 * 1024
	}
	return int64(fileSize)
}

// 上传文件资源路径
func uploadFileDir() string {
	fileDir := fmt.Sprint(config.Get("upload.fileDir"))
	if fileDir == "" || fileDir == "<nil>" {
		fileDir = "/tmp"
	}
	return fileDir
}

// 文件上传扩展名白名单
func uploadWhiteList() []string {
	arr := config.Get("upload.whitelist").([]any)
	strings := make([]string, len(arr))
	for i, val := range arr {
		if str, ok := val.(string); ok {
			strings[i] = str
		}
	}
	return strings
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
func isAllowWrite(fileName string, allowExts []string, fileSize int64) error {
	// 判断上传文件名称长度
	if len(fileName) > DEFAULT_FILE_NAME_LENGTH {
		return fmt.Errorf("the maximum length limit for uploading file names is %d", DEFAULT_FILE_NAME_LENGTH)
	}

	// 最大上传文件大小
	maxFileSize := uploadFileSize()
	if fileSize > maxFileSize {
		return fmt.Errorf("maximum upload file size %s", parse.Bit(float64(maxFileSize)))
	}

	// 判断文件拓展是否为允许的拓展类型
	fileExt := filepath.Ext(fileName)
	hasExt := false
	if len(allowExts) == 0 {
		allowExts = uploadWhiteList()
	}
	for _, ext := range allowExts {
		if ext == fileExt {
			hasExt = true
			break
		}
	}
	if !hasExt {
		return fmt.Errorf("the upload file type is not supported, only the following types are supported: %s", strings.Join(allowExts, ","))
	}

	return nil
}

// 检查文件允许本地读取
//
// filePath 文件存放资源路径，URL相对地址
func isAllowRead(filePath string) error {
	// 禁止目录上跳级别
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("prohibit jumping levels on the directory")
	}

	// 检查允许下载的文件规则
	fileExt := filepath.Ext(filePath)
	hasExt := false
	for _, ext := range uploadWhiteList() {
		if ext == fileExt {
			hasExt = true
			break
		}
	}
	if !hasExt {
		return fmt.Errorf("rules for illegally downloaded files: %s", fileExt)
	}

	return nil
}

// TransferUploadFile 上传资源文件转存
//
// allowExts 允许上传拓展类型（含“.”)，如 ['.png','.jpg']
func TransferUploadFile(file *multipart.FileHeader, allowExts []string) (string, error) {
	// 上传文件检查
	err := isAllowWrite(file.Filename, allowExts, file.Size)
	if err != nil {
		return "", err
	}
	// 上传资源路径
	dir := uploadFileDir()
	// 新文件名称并组装文件地址
	filePath := date.ParseDatePath(time.Now())
	fileName := generateFileName(file.Filename)
	writePathFile := filepath.Join(dir, filePath, fileName)
	// 存入新文件路径
	err = transferToNewFile(file, writePathFile)
	if err != nil {
		return "", err
	}
	return writePathFile, nil
}

// ReadUploadFileStream 上传资源文件读取
//
// filePath 文件存放资源路径，URL相对地址 如：/upload/common/2023/06/xxx.png
//
// headerRange 断点续传范围区间，bytes=0-12131
func ReadUploadFileStream(fileAsbPath, headerRange string) (map[string]any, error) {
	// 读取文件检查
	err := isAllowRead(fileAsbPath)
	if err != nil {
		return map[string]any{}, err
	}

	// 响应结果
	result := map[string]any{
		"range":     "",
		"chunkSize": 0,
		"fileSize":  0,
		"data":      []byte{},
	}

	// 文件大小
	fileSize := getFileSize(fileAsbPath)
	if fileSize <= 0 {
		return result, fmt.Errorf("file does not exist")
	}
	result["fileSize"] = fileSize

	if headerRange != "" {
		partsStr := strings.Replace(headerRange, "bytes=", "", 1)
		parts := strings.Split(partsStr, "-")
		start, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start > fileSize {
			start = 0
		}
		end, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end > fileSize {
			end = fileSize - 1
		}
		if start > end {
			start = end
		}

		// 分片结果
		result["range"] = fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize)
		result["chunkSize"] = end - start + 1
		byteArr, err := getFileStream(fileAsbPath, start, end)
		if err != nil {
			return map[string]any{}, fmt.Errorf("fail to read file")
		}
		result["data"] = byteArr
		return result, nil
	}

	byteArr, err := getFileStream(fileAsbPath, 0, fileSize)
	if err != nil {
		return map[string]any{}, fmt.Errorf("fail to read file")
	}
	result["data"] = byteArr
	return result, nil
}

// TransferChunkUploadFile 上传资源切片文件转存
//
// file 上传文件对象
//
// index 切片文件序号
//
// identifier 切片文件目录标识符
func TransferChunkUploadFile(file *multipart.FileHeader, index, identifier string) (string, error) {
	// 上传文件检查
	err := isAllowWrite(file.Filename, []string{}, file.Size)
	if err != nil {
		return "", err
	}
	// 上传资源路径
	dir := uploadFileDir()
	// 新文件名称并组装文件地址
	filePath := date.ParseDatePath(time.Now())
	writePathFile := path.Join(dir, filePath, "chunk", identifier, index)
	// 存入新文件路径
	err = transferToNewFile(file, writePathFile)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(writePathFile), nil
}

// 上传资源切片文件检查
//
// identifier 切片文件目录标识符
//
// originalFileName 原始文件名称，如logo.png
func ChunkCheckFile(identifier, originalFileName string) ([]string, error) {
	// 读取文件检查
	err := isAllowWrite(originalFileName, []string{}, 0)
	if err != nil {
		return []string{}, err
	}
	// 上传资源路径
	dir := uploadFileDir()
	// 切片存放目录
	filePath := date.ParseDatePath(time.Now())
	readPath := path.Join(dir, filePath, "chunk", identifier)
	fileList, err := getDirFileNameList(readPath)
	if err != nil {
		return []string{}, fmt.Errorf("fail to read file")
	}
	return fileList, nil
}

// 上传资源切片文件检查
//
// identifier 切片文件目录标识符
//
// originalFileName 原始文件名称，如logo.png
func ChunkMergeFile(identifier, originalFileName string) (string, error) {
	// 读取文件检查
	err := isAllowWrite(originalFileName, []string{}, 0)
	if err != nil {
		return "", err
	}
	// 上传资源路径
	dir := uploadFileDir()
	// 新文件名称并组装文件地址
	filePath := date.ParseDatePath(time.Now())
	fileName := generateFileName(originalFileName)
	writePathFile := filepath.Join(dir, filePath, fileName)
	// 切片存放目录
	readPath := path.Join(dir, filePath, "chunk", identifier)
	err = mergeToNewFile(readPath, writePathFile)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(writePathFile), nil
}
