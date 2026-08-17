package file

import (
	"fmt"
	"io"
	"math"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tsmask/go-oam/pkg/generate"
)

// ==================== 配置 ====================

// FileConfig 文件上传配置
type FileConfig struct {
	Dir        string   // 存储目录，必填
	MaxSize    int      // 最大文件大小 (MB)，小于1默认1MB
	AllowExts  []string // 允许的扩展名，如 [".jpg", ".png"]，为空则不限制
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
	const bytesPerMB = int64(1024 * 1024)
	if size > math.MaxInt64/bytesPerMB {
		return math.MaxInt64
	}
	return size * 1024 * 1024
}

// withUploadLock serializes upload operations for one storage directory.
func (c FileConfig) withUploadLock(fn func() error) error {
	dir := c.Dir
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return withPathLock(filepath.Join(dir, ".upload-lock"), fn)
}

// ==================== 上传对象 ====================

// FileUpload 文件上传对象。
// 单文件上传：设置 Name + (Reader 或 Data)，调用 Save。
// 分片上传：设置 Id + Index + (Reader 或 Data)，调用 ChunkSave。
// Reader 优先于 Data，适合大文件流式上传，内存占用恒定。
type FileUpload struct {
	Name   string    // 文件名（单文件上传时必填）
	Reader io.Reader // 文件数据流，优先使用，适合大文件（与 Data 二选一）
	Data   *[]byte   // 文件数据，由调用方通过 io.ReadAll 读取后传入（小文件场景）
	Id     string    // 分片标识，同一文件的所有分片共享（分片上传时必填）
	Index  string    // 分片序号，从 "0" 开始递增（分片上传时必填）
}

// Save 校验并保存单文件，返回存储路径。
// 校验内容：文件名长度、文件大小、允许的扩展名。
// Reader 优先：流式写入临时文件，超过限制自动清理；Data 模式全量写入。
func (u FileUpload) Save(cfg FileConfig) (string, error) {
	if u.Name == "" {
		return "", fmt.Errorf("upload file name is empty")
	}
	if u.Reader == nil && u.Data == nil {
		return "", fmt.Errorf("upload file %s has no Reader or Data", u.Name)
	}

	var dst string
	err := cfg.withUploadLock(func() error {
		if err := checkNameAndExt(cfg, u.Name); err != nil {
			return err
		}
		dst = genPath(cfg.Dir, u.Name)

		if u.Reader != nil {
			return streamToPath(dst, u.Reader, cfg.maxSizeBytes())
		}

		data := *u.Data
		limit := cfg.maxSizeBytes()
		if int64(len(data)) > limit {
			return fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(data), limit)
		}
		return writeBytes(dst, data)
	})
	if err != nil {
		return "", err
	}
	return canonicalPath(dst), nil
}

// ChunkSave 校验并保存分片数据，存储为 {dir}/{id}/{index}。
// 校验内容：分片序号、分片大小不超过 MaxSize。
// Reader 优先：流式写入临时文件，超过限制自动清理；Data 模式全量写入。
func (u FileUpload) ChunkSave(cfg FileConfig) error {
	index, err := parseChunkIndex(u.Index)
	if err != nil {
		return err
	}
	id, err := uploadIDPath(cfg, u.Id)
	if err != nil {
		return err
	}
	if u.Reader == nil && u.Data == nil {
		return fmt.Errorf("chunk %s/%d has no Reader or Data", u.Id, index)
	}

	return cfg.withUploadLock(func() error {
		dst := filepath.Join(id, strconv.FormatUint(index, 10))
		if u.Reader != nil {
			return streamToPath(dst, u.Reader, cfg.maxSizeBytes())
		}

		data := *u.Data
		limit := cfg.maxSizeBytes()
		if int64(len(data)) > limit {
			return fmt.Errorf("chunk size %d exceeds limit %d", len(data), limit)
		}
		return writeBytes(dst, data)
	})
}

// ChunkList 查询已上传的分片文件名列表（按分片序号排序）。
func (c FileConfig) ChunkList(id string) ([]string, error) {
	chunkDir, err := uploadIDPath(c, id)
	if err != nil {
		return nil, err
	}

	var names []string
	err = c.withUploadLock(func() error {
		var listErr error
		var chunks []chunkFile
		chunks, listErr = readChunkFiles(chunkDir)
		for _, chunk := range chunks {
			names = append(names, chunk.name)
		}
		return listErr
	})
	return names, err
}

// ChunkMerge 合并分片为完整文件，按分片序号升序拼接。
// 合并成功后自动删除分片目录，返回合并后的文件存储路径。
func (c FileConfig) ChunkMerge(id string, fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("merged file name is empty")
	}
	if err := checkNameAndExt(c, fileName); err != nil {
		return "", err
	}
	chunkDir, err := uploadIDPath(c, id)
	if err != nil {
		return "", err
	}

	var dst string
	err = c.withUploadLock(func() error {
		dst = genPath(c.Dir, fileName)
		return mergeFiles(chunkDir, dst)
	})
	if err != nil {
		return "", err
	}
	return canonicalPath(dst), nil
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
	if total == 0 {
		return FileRange{Data: []byte{}, Start: 0, End: -1, Total: 0}, nil
	}

	// 修正边界
	if start < 0 {
		start = 0
	}
	if end <= 0 || end >= total {
		end = total - 1
	}
	if start >= total {
		return FileRange{Data: []byte{}, Start: total, End: total - 1, Total: total}, nil
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

// reIllegal 非法文件名字符 + 空白，统一替换为 _ 号
var reIllegal = regexp.MustCompile(`[\\/:*?"<>|\s]+`)

// genPath 生成存储路径：{dir}/{清理名}_{随机码}.{ext}
func genPath(dir, fileName string) string {
	normalized := strings.ReplaceAll(fileName, `\`, `/`)
	ext := pathpkg.Ext(normalized)
	base := strings.TrimSuffix(normalized, ext)

	base = reIllegal.ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "file"
	}
	ext = sanitizeExtension(ext)

	return filepath.Join(dir, base+"_"+generate.Code(6)+ext)
}

func sanitizeExtension(ext string) string {
	ext = reIllegal.ReplaceAllString(ext, "_")
	ext = strings.TrimRight(ext, " .")
	if ext == "." {
		return ""
	}
	return ext
}

// checkNameAndExt 校验文件名长度和允许的扩展名
func checkNameAndExt(cfg FileConfig, name string) error {
	if len(name) > cfg.maxNameLen() {
		return fmt.Errorf("file name length exceeds %d characters", cfg.maxNameLen())
	}
	if len(cfg.AllowExts) > 0 {
		ext := strings.ToLower(filepath.Ext(name))
		allowed := slices.ContainsFunc(cfg.AllowExts, func(allowedExt string) bool {
			return strings.EqualFold(allowedExt, ext)
		})
		if !allowed {
			return fmt.Errorf("unsupported file extension %s", ext)
		}
	}
	return nil
}

func uploadIDPath(cfg FileConfig, id string) (string, error) {
	if !validUploadID(id) {
		return "", fmt.Errorf("invalid upload id %q", id)
	}
	dir := cfg.Dir
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return filepath.Join(dir, id), nil
}

func validUploadID(id string) bool {
	if id == "" || id == "." || id == ".." || !utf8.ValidString(id) || reIllegal.MatchString(id) {
		return false
	}
	for _, r := range id {
		if r < 32 || r == 0x7f || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func parseChunkIndex(index string) (uint64, error) {
	parsed, err := strconv.ParseUint(index, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid chunk index %q: %w", index, err)
	}
	return parsed, nil
}

// streamToPath 从 Reader 流式写入目标路径，超过 limit 字节则报错并清理临时文件。
// 使用临时文件 + rename 保证原子性。
func streamToPath(dst string, src io.Reader, limit int64) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0775); err != nil {
		return err
	}
	return writeFileAtomic(dst, 0644, func(f *os.File) error {
		n, err := io.Copy(f, io.LimitReader(src, limit+1))
		if err == nil && n > limit {
			return fmt.Errorf("file size exceeds limit %d bytes", limit)
		}
		return err
	})
}

// writeBytes 原子写入字节数据到目标路径，自动创建父目录
func writeBytes(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0775); err != nil {
		return err
	}
	return writeFileAtomic(dst, 0644, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

type chunkFile struct {
	name  string
	index uint64
}

// mergeFiles 按数值序号合并分片目录中的文件到 writePath。
// 合并成功后删除分片目录；失败时保留分片且不产生不完整输出。
func mergeFiles(chunkDir, writePath string) error {
	files, err := readChunkFiles(chunkDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no chunk files found in %s", chunkDir)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].index < files[j].index
	})

	if err := os.MkdirAll(filepath.Dir(writePath), 0775); err != nil {
		return err
	}
	err = writeFileAtomic(writePath, 0644, func(out *os.File) error {
		for _, chunk := range files {
			src, openErr := os.Open(filepath.Join(chunkDir, chunk.name))
			if openErr != nil {
				return openErr
			}
			_, copyErr := io.Copy(out, src)
			closeErr := src.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := os.RemoveAll(chunkDir); err != nil {
		return fmt.Errorf("remove chunk dir: %w", err)
	}
	return nil
}

// readChunkFiles returns regular files in a chunk directory in index order.
func readChunkFiles(dirPath string) ([]chunkFile, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	files := make([]chunkFile, 0, len(entries))
	seenIndexes := make(map[uint64]struct{}, len(entries))
	for _, e := range entries {
		if e.Type().IsRegular() {
			index, parseErr := parseChunkIndex(e.Name())
			if parseErr != nil {
				return nil, fmt.Errorf("read chunk dir: %w", parseErr)
			}
			if _, exists := seenIndexes[index]; exists {
				return nil, fmt.Errorf("duplicate chunk index %d in %s", index, dirPath)
			}
			seenIndexes[index] = struct{}{}
			files = append(files, chunkFile{name: e.Name(), index: index})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].index < files[j].index
	})
	return files, nil
}
