//go:build !windows
// +build !windows

package file

import (
	"fmt"
	"os"
	"os/user"
	"syscall"
)

// getFileInfo 获取系统特定的文件信息s
func getFileInfo(info os.FileInfo) (linkCount int64, owner, group string) {
	// Unix-like 系统 (Linux, macOS)
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		// 获取用户名
		ownerName := "root"
		if stat.Uid != 0 {
			if u, err := user.LookupId(fmt.Sprint(stat.Uid)); err == nil {
				ownerName = u.Username
			}
		}

		// 获取组名
		groupName := "root"
		if stat.Gid != 0 {
			if g, err := user.LookupGroupId(fmt.Sprint(stat.Gid)); err == nil {
				groupName = g.Name
			}
		}

		return int64(stat.Nlink), ownerName, groupName
	}
	return 1, "", ""
}
