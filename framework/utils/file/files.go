package file

import (
	"os"
	"path/filepath"
	"sort"
)

// SystemFileListRow 文件列表行数据
type SystemFileListRow struct {
	FileType     string `json:"fileType"`     // 文件类型 dir, file, symlink
	FileMode     string `json:"fileMode"`     // 文件的权限
	LinkCount    int64  `json:"linkCount"`    // 硬链接数目
	Owner        string `json:"owner"`        // 所属用户
	Group        string `json:"group"`        // 所属组
	Size         int64  `json:"size"`         // 文件的大小
	ModifiedTime int64  `json:"modifiedTime"` // 最后修改时间，单位为秒
	FileName     string `json:"fileName"`     // 文件的名称
}

// SystemFileList 获取系统文件列表
// search 文件名后模糊*
//
// return 行记录，异常
func SystemFileList(path, search string) ([]SystemFileListRow, error) {
	var rows []SystemFileListRow

	// 构建搜索模式
	pattern := "*"
	if search != "" {
		pattern = search + pattern
	}

	// 读取目录内容
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// 遍历目录项
	for _, entry := range entries {
		// 匹配文件名
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil || !matched {
			continue
		}

		// 获取文件详细信息
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 确定文件类型
		fileType := "file"
		if info.IsDir() {
			fileType = "dir"
		} else if info.Mode()&os.ModeSymlink != 0 {
			fileType = "symlink"
		}

		// 获取系统特定的文件信息
		linkCount, owner, group := getFileInfo(info)

		// 组装文件信息
		rows = append(rows, SystemFileListRow{
			FileMode:     info.Mode().String(),
			FileType:     fileType,
			LinkCount:    linkCount,
			Owner:        owner,
			Group:        group,
			Size:         info.Size(),
			ModifiedTime: info.ModTime().UnixMilli(),
			FileName:     entry.Name(),
		})
	}

	// 按时间排序
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].ModifiedTime > rows[j].ModifiedTime
	})
	return rows, nil
}
