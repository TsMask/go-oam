package generate

import (
	"math/rand"
	"time"

	"github.com/tsmask/go-oam/src/framework/logger"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// 生成随机Code
// 包含数字、小写字母
// 不保证长度满足
func Code(size int) string {
	str, err := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", size)
	if err != nil {
		logger.Infof("%d : %v", size, err)
		return ""
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
	return str
}

// 随机数 纯数字0-9
func Number(size int) int {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	min := int64(0)
	max := int64(9 * int(size))
	return int(random.Int63n(max-min+1) + min)
}
