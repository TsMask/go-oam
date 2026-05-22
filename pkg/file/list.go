package file

import (
	"os"
	"path/filepath"
	"sort"
)

// FileEntry 文件条目信息
type FileEntry struct {
	Name    string `json:"name"`    // 文件名
	Path    string `json:"path"`    // 完整路径
	Type    string `json:"type"`    // 类型: dir, file, symlink
	Mode    string `json:"mode"`    // 权限字符串如 "rwxr-xr-x"
	Size    int64  `json:"size"`    // 文件大小(字节)
	ModTime int64  `json:"modTime"` // 最后修改时间(Unix毫秒)
	Owner   string `json:"owner"`   // 所属用户
	Group   string `json:"group"`   // 所属组
	Links   int64  `json:"links"`   // 硬链接数
}

// ListDir 列出当前目录下的文件条目，按修改时间倒序排列。
// pattern 为 glob 匹配模式（如 "*.log"），传空字符串匹配所有。
//
//	entries, err := file.ListDir("/var/log", "")
//	entries, err := file.ListDir("/var/log", "*.log")
func ListDir(dirPath string, pattern string) ([]FileEntry, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var results []FileEntry
	for _, de := range dirEntries {
		if pattern != "" {
			matched, matchErr := filepath.Match(pattern, de.Name())
			if matchErr != nil || !matched {
				continue
			}
		}

		info, err := de.Info()
		if err != nil {
			continue
		}

		results = append(results, newFileEntry(info, de.Name(), dirPath))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ModTime > results[j].ModTime
	})
	return results, nil
}

// Stat 获取单个文件的详细信息
func Stat(filePath string) (FileEntry, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return FileEntry{}, err
	}
	return newFileEntry(info, filepath.Base(filePath), filepath.Dir(filePath)), nil
}

// Exists 检查文件或目录是否存在
func Exists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// newFileEntry 从 os.FileInfo 构造 FileEntry
func newFileEntry(info os.FileInfo, name, dirPath string) FileEntry {
	fileType := "file"
	if info.IsDir() {
		fileType = "dir"
	} else if info.Mode()&os.ModeSymlink != 0 {
		fileType = "symlink"
	}

	links, owner, group := fileInfoExtra(info)

	return FileEntry{
		Name:    name,
		Path:    filepath.Join(dirPath, name),
		Type:    fileType,
		Mode:    info.Mode().String(),
		Size:    info.Size(),
		ModTime: info.ModTime().UnixMilli(),
		Owner:   owner,
		Group:   group,
		Links:   links,
	}
}
