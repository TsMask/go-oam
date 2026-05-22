package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tsmask/go-oam/pkg/generate"
)

// ==================== 配置 ====================

// FileConfig 文件上传配置
type FileConfig struct {
	Dir        string   // 存储目录，必填
	MaxSize    int      // 最大文件大小 (MB)，小于1默认1MB
	WhiteExts  []string // 扩展名白名单，如 [".jpg", ".png"]，为空则不限制
	MaxNameLen int      // 文件名最大长度，0默认100
}

// maxNameLen 获取文件名最大长度，未设置时默认 100
func (c FileConfig) maxNameLen() int {
	if c.MaxNameLen > 0 {
		return c.MaxNameLen
	}
	return 100
}

// maxSizeBytes 最大文件大小换算为字节，未设置时默认 1MB
func (c FileConfig) maxSizeBytes() int64 {
	size := int64(c.MaxSize)
	if size < 1 {
		size = 1
	}
	return size * 1024 * 1024
}

// ==================== 上传对象 ====================

// FileUpload 文件上传对象。
// 单文件上传：设置 Name + Data，调用 Save。
// 分片上传：设置 Id + Index + Data，调用 ChunkSave。
type FileUpload struct {
	Name  string  // 文件名（单文件上传时必填）
	Data  *[]byte // 文件数据，由调用方通过 io.ReadAll 读取后传入
	Id    string  // 分片标识，同一文件的所有分片共享（分片上传时必填）
	Index string  // 分片序号，从 "0" 开始递增（分片上传时必填）
}

// Save 校验并保存单文件，返回存储路径。
// 校验内容：文件名长度、文件大小、扩展名白名单。
func (u FileUpload) Save(cfg FileConfig) (string, error) {
	data := *u.Data
	if err := checkUpload(cfg, u.Name, int64(len(data))); err != nil {
		return "", err
	}
	dst := genPath(cfg.Dir, u.Name)
	return dst, writeBytes(dst, data)
}

// ChunkSave 校验并保存分片数据，存储为 {dir}/{id}/{index}。
// 校验内容：分片大小不超过 MaxSize。
func (u FileUpload) ChunkSave(cfg FileConfig) error {
	data := *u.Data
	limit := cfg.maxSizeBytes()
	if int64(len(data)) > limit {
		return fmt.Errorf("chunk size %d exceeds limit %d", len(data), limit)
	}
	return writeBytes(filepath.Join(cfg.Dir, u.Id, u.Index), data)
}

// ChunkList 查询已上传的分片文件名列表（按文件名排序）。
func (c FileConfig) ChunkList(id string) ([]string, error) {
	return listRegularFiles(filepath.Join(c.Dir, id))
}

// ChunkMerge 合并分片为完整文件，按分片序号升序拼接。
// 合并成功后自动删除分片目录，返回合并后的文件存储路径。
func (c FileConfig) ChunkMerge(id string, fileName string) (string, error) {
	chunkDir := filepath.Join(c.Dir, id)
	dst := genPath(c.Dir, fileName)
	return dst, mergeFiles(chunkDir, dst)
}

// ==================== 文件读取 ====================

// FileRange 文件范围读取结果
type FileRange struct {
	Data  []byte `json:"data"`  // 读取的数据
	Start int64  `json:"start"` // 起始偏移（字节）
	End   int64  `json:"end"`   // 结束偏移（字节）
	Total int64  `json:"total"` // 文件总大小（字节）
}

// ReadStream 读取文件指定范围数据，适用于 HTTP Range 分片下载。
// start/end 为字节偏移量，end 传 0 表示读到文件末尾。
func ReadStream(filePath string, start, end int64) (FileRange, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return FileRange{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return FileRange{}, err
	}
	total := info.Size()

	// 修正边界
	if start < 0 {
		start = 0
	}
	if end <= 0 || end >= total {
		end = total - 1
	}
	if start > end {
		start = end
	}

	size := end - start + 1
	buf := make([]byte, size)
	n, err := f.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return FileRange{}, err
	}

	return FileRange{Data: buf[:n], Start: start, End: end, Total: total}, nil
}

// ==================== 内部辅助 ====================

// reIllegal 非法文件名字符 + 空白，统一替换为 * 号
var reIllegal = regexp.MustCompile(`[\\/:*?"<>|\s]+`)

// checkUpload 校验上传约束：文件名长度、文件大小、扩展名白名单
func checkUpload(cfg FileConfig, name string, size int64) error {
	if len(name) > cfg.maxNameLen() {
		return fmt.Errorf("file name length exceeds %d characters", cfg.maxNameLen())
	}
	if size > cfg.maxSizeBytes() {
		return fmt.Errorf("file size %d bytes exceeds limit %d bytes", size, cfg.maxSizeBytes())
	}
	if len(cfg.WhiteExts) > 0 {
		ext := strings.ToLower(filepath.Ext(name))
		if !slices.Contains(cfg.WhiteExts, ext) {
			return fmt.Errorf("unsupported file extension %s", ext)
		}
	}
	return nil
}

// genPath 生成存储路径：{dir}/{清理名}_{随机码}.{ext}
// 非法字符和空格替换为 *，首尾 * 去除
func genPath(dir, fileName string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	base = reIllegal.ReplaceAllString(base, "*")
	base = strings.Trim(base, "*")
	if base == "" {
		base = "file"
	}
	return filepath.Join(dir, base+"_"+generate.Code(6)+ext)
}

// writeBytes 写入字节数据到目标路径，自动创建父目录
func writeBytes(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0775); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// mergeFiles 按数值序号合并分片目录中的文件到 writePath。
// 合并完成后删除分片目录（无论成功失败）。
func mergeFiles(chunkDir, writePath string) error {
	names, err := listRegularFiles(chunkDir)
	if err != nil {
		return fmt.Errorf("read chunk dir: %w", err)
	}
	if len(names) == 0 {
		return fmt.Errorf("no chunk files found in %s", chunkDir)
	}

	// 按文件名数值排序，确保分片顺序正确
	sort.Slice(names, func(i, j int) bool {
		ni, _ := strconv.Atoi(names[i])
		nj, _ := strconv.Atoi(names[j])
		return ni < nj
	})

	if err = os.MkdirAll(filepath.Dir(writePath), 0775); err != nil {
		return err
	}

	out, err := os.Create(writePath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}

	// 逐个分片流式追加到输出文件
	var mergeErr error
	for _, name := range names {
		src, openErr := os.Open(filepath.Join(chunkDir, name))
		if openErr != nil {
			mergeErr = openErr
			break
		}
		_, mergeErr = io.Copy(out, src)
		src.Close()
		if mergeErr != nil {
			break
		}
	}

	if closeErr := out.Close(); mergeErr == nil {
		mergeErr = closeErr
	}

	// 清理分片目录
	os.RemoveAll(chunkDir)
	return mergeErr
}

// listRegularFiles 列出目录下的常规文件名（不含子目录），按文件名排序
func listRegularFiles(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// openTmp 创建临时文件用于原子写入，调用方负责 Close + Rename。
func openTmp(targetPath string) (*os.File, error) {
	return os.OpenFile(targetPath+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
}

// copyToFile 将 src 流式写入 dstPath，使用预分配 buf。
func copyToFile(dstPath string, src io.Reader, buf []byte) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.CopyBuffer(dst, src, buf)
	return err
}
