package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// SHA256ToBase64 编码字符串
func SHA256ToBase64(str string) string {
	hash := sha256.Sum256([]byte(str))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// SHA256Hmac HMAC-SHA256算法
func SHA256Hmac(key string, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// MD5 md5加密
func MD5(str string) (md5str string) {
	data := []byte(str)
	has := md5.Sum(data)
	md5str = fmt.Sprintf("%x", has)
	return md5str
}
