package parse

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// IsText 判断字节切片是否为文本。
// 空切片返回 true；非 UTF-8 或含控制字符（\n \r \t 除外）返回 false。
func IsText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if !utf8.Valid(data) {
		return false
	}
	for _, r := range string(data) {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

// Number 解析数值型，支持 string/int/uint/float/bool，无法解析返回 0。
func Number(value any) int64 {
	switch v := value.(type) {
	case string:
		if v == "" {
			return 0
		}
		num, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return num
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int()
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(v).Uint())
	case float32, float64:
		return int64(reflect.ValueOf(v).Float())
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// Boolean 解析布尔型，支持 string/int/uint/float/bool，无法解析返回 false。
func Boolean(value any) bool {
	switch v := value.(type) {
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return b
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() != 0
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0
	case bool:
		return v
	default:
		return false
	}
}

// Bit 将字节数格式化为人类可读的容量字符串（1024 进制）。
// 例如 Bit(1023) -> "1023.00 B"，Bit(1024) -> "1.00 KB"。
func Bit(bit float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	for i := 0; i < len(units); i++ {
		if bit < 1024 || i == len(units)-1 {
			return fmt.Sprintf("%.2f %s", bit, units[i])
		}
		bit /= 1024
	}
	return fmt.Sprintf("%.2f B", bit)
}

// SafeContent 对敏感内容进行掩码处理，支持多字节 UTF-8 字符。
// 显示首尾少量字符，中间以 * 替代。
//
//	长度 < 3:  全部掩码       "ab"    -> "**"
//	长度 < 6:  保留首字符     "abc"   -> "a**"
//	长度 < 10: 保留首尾字符   "abcdef" -> "a****f"
//	长度 < 15: 保留首尾各2    "abcdefghij" -> "ab******ij"
//	长度 >= 15: 保留首尾各3   "abcdefghijklmno" -> "abc*********o"
func SafeContent(value string) string {
	runes := []rune(value)
	length := len(runes)
	switch {
	case length < 3:
		return strings.Repeat("*", length)
	case length < 6:
		return string(runes[0]) + strings.Repeat("*", length-1)
	case length < 10:
		return string(runes[0]) + strings.Repeat("*", length-2) + string(runes[length-1])
	case length < 15:
		return string(runes[:2]) + strings.Repeat("*", length-4) + string(runes[length-2:])
	default:
		return string(runes[:3]) + strings.Repeat("*", length-6) + string(runes[length-3:])
	}
}

// ConvertIPMask 将 CIDR 前缀长度转为点分十进制子网掩码。
// 例如 24 -> "255.255.255.0"，20 -> "255.255.240.0"。
// bits 超出 [0,32] 范围返回 "255.255.255.255"。
func ConvertIPMask(bits int64) string {
	if bits < 0 || bits > 32 {
		return "255.255.255.255"
	}
	// 显式使用 uint32 运算，避免 32 位平台上 int 溢出
	var mask uint32
	if bits == 0 {
		mask = 0
	} else {
		mask = (uint32(1)<<uint(bits) - 1) << (32 - uint(bits))
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		mask>>24,
		(mask>>16)&0xFF,
		(mask>>8)&0xFF,
		mask&0xFF,
	)
}
