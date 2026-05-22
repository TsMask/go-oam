package generate

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	digits       = "0123456789"
	lowerAlpha   = "abcdefghijklmnopqrstuvwxyz"
	upperAlpha   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphanumeric = digits + lowerAlpha + upperAlpha
	codeAlpha    = digits + lowerAlpha
)

// Code 生成指定长度的随机编码，仅包含数字和小写字母。
// 使用 crypto/rand 保证密码学安全。
func Code(size int) string {
	return randomString(codeAlpha, size)
}

// String 生成指定长度的随机字符串，包含数字、大小写字母。
// 使用 crypto/rand 保证密码学安全。
func String(size int) string {
	return randomString(alphanumeric, size)
}

// Number 生成指定位数的随机正整数，首位保证非零。
// size 范围 [1,18]，超出范围自动截断；size <= 0 返回 0。
func Number(size int) int64 {
	if size < 1 {
		return 0
	}
	if size > 18 {
		size = 18
	}
	// [10^(size-1), 10^size - 1]，保证恰好 size 位且首位非零
	lo := int64(1)
	for i := 1; i < size; i++ {
		lo *= 10
	}
	hi := lo*10 - 1
	n, err := rand.Int(rand.Reader, big.NewInt(hi-lo+1))
	if err != nil {
		return lo
	}
	return lo + n.Int64()
}

// randomString 从 alphabet 中安全随机选取 size 个字符组成字符串。
func randomString(alphabet string, size int) string {
	if size <= 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.Grow(size)
	max := big.NewInt(int64(len(alphabet)))
	for range size {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			sb.WriteByte(alphabet[0])
			continue
		}
		sb.WriteByte(alphabet[n.Int64()])
	}
	return sb.String()
}
