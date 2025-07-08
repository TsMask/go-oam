package cmd

import (
	"os/exec"
	"strings"
)

// CheckIllegal 检查传入的字符串参数中是否包含一些特殊字符
func CheckIllegal(args ...string) bool {
	if args == nil {
		return false
	}
	illegalChars := []string{"&", "|", ";", "$", "'", "`", "(", ")", "\""}
	for _, arg := range args {
		for _, char := range illegalChars {
			if strings.Contains(arg, char) {
				return true
			}
		}
	}
	return false
}

// HasNoPasswordSudo 检查当前用户是否拥有sudo权限
func HasNoPasswordSudo() bool {
	cmd2 := exec.Command("sudo", "-n", "uname")
	err2 := cmd2.Run()
	return err2 == nil
}

// SudoHandleCmd 是否拥有sudo权限并拼接
func SudoHandleCmd() string {
	cmd := exec.Command("sudo", "-n", "uname")
	if err := cmd.Run(); err == nil {
		return "sudo "
	}
	return ""
}

// Which 可执行文件是否在系统的PATH环境变量中
func Which(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
