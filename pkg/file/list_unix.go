//go:build !windows
// +build !windows

package file

import (
	"fmt"
	"os"
	"os/user"
	"sync"
	"syscall"
)

// uid/gid 缓存，避免同一 uid/gid 重复调用 user.LookupId/user.LookupGroupId
var (
	uidCache sync.Map // uint32 -> string
	gidCache sync.Map // uint32 -> string
)

// lookupUsername 根据 uid 查询用户名，结果缓存
func lookupUsername(uid uint32) string {
	if v, ok := uidCache.Load(uid); ok {
		return v.(string)
	}
	name := "root"
	if uid != 0 {
		if u, err := user.LookupId(fmt.Sprint(uid)); err == nil {
			name = u.Username
		}
	}
	uidCache.Store(uid, name)
	return name
}

// lookupGroupname 根据 gid 查询组名，结果缓存
func lookupGroupname(gid uint32) string {
	if v, ok := gidCache.Load(gid); ok {
		return v.(string)
	}
	name := "root"
	if gid != 0 {
		if g, err := user.LookupGroupId(fmt.Sprint(gid)); err == nil {
			name = g.Name
		}
	}
	gidCache.Store(gid, name)
	return name
}

// fileInfoExtra 从 os.FileInfo 提取硬链接数、所属用户、所属组
func fileInfoExtra(info os.FileInfo) (links int64, owner, group string) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 1, "", ""
	}
	return int64(stat.Nlink), lookupUsername(stat.Uid), lookupGroupname(stat.Gid)
}
