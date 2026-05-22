//go:build windows
// +build windows

package file

import "os"

// fileInfoExtra Windows 平台返回默认值
func fileInfoExtra(_ os.FileInfo) (links int64, owner, group string) {
	return 1, "Administrator", "Administrators"
}
