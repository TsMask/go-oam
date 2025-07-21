//go:build windows
// +build windows

package file

import (
	"os"
)

// getFileInfo 获取系统特定的文件信息
func getFileInfo(_ os.FileInfo) (linkCount int64, owner, group string) {
	return 1, "Administrator", "Administrators"
}
