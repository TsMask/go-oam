package generate

import (
	"fmt"
	"strconv"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// 生成随机Code
// 包含数字、小写字母
// 不保证长度满足
func Code(size int) string {
	str, err := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", size)
	if err != nil {
		return ""
	}
	// 位数不足时向后补 e，直至达到指定长度
	for len(str) < size {
		str += "e"
	}
	return str
}

// 生成随机字符串
// 包含数字、大小写字母、下划线、横杠
// 不保证长度满足
func String(size int) string {
	str, err := gonanoid.New(size)
	if err != nil {
		return ""
	}
	// 位数不足时向后补 e，直至达到指定长度
	for len(str) < size {
		str += "e"
	}
	return str
}

// 生成随机整数值 size最大18
func Number(size int) int64 {
	// int64 最大值为 9223372036854775807，共 19 位
	maxSize := 18
	if size > maxSize {
		size = maxSize
	}

	str, err := gonanoid.Generate("1234567890", size)
	if err != nil {
		return 0
	}

	// 位数不足时向后补 0
	if len(str) < size {
		str = fmt.Sprintf("%-*s", size, str)
		str = str[:size] // 确保长度准确
	}

	if strings.HasPrefix(str, "0") {
		str = strings.Replace(str, "0", "1", 1)
	}

	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
