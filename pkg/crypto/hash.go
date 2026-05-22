package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
)

// MD5 计算字符串的 MD5 哈希值，返回 hex 编码
func MD5(data string) string {
	sum := md5.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

// SHA256 计算字符串的 SHA-256 哈希值，返回 hex 编码
func SHA256(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// HMACSHA256 计算 HMAC-SHA256 签名，返回 hex 编码
func HMACSHA256(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
