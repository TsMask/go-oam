package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// AESEncryptBase64 AES-CBC 加密并编码为 Base64 字符串
// key 长度必须为 16/24/32 字节
func AESEncryptBase64(text, key string) (string, error) {
	if text == "" {
		return "", nil
	}
	ciphertext, err := AESEncrypt([]byte(text), []byte(key))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecryptBase64 解码 Base64 字符串并进行 AES-CBC 解密
// key 长度必须为 16/24/32 字节
func AESDecryptBase64(text, key string) (string, error) {
	if text == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	plaintext, err := AESDecrypt(ciphertext, []byte(key))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// AESEncrypt AES-CBC 加密（PKCS7 填充）
// 密文结构: IV (16B) + 密文数据
func AESEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}

	plaintext = pkcs7Pad(plaintext, aes.BlockSize)

	// IV + 密文
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("generate iv: %w", err)
	}

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

// AESDecrypt AES-CBC 解密（PKCS7 去填充）
func AESDecrypt(ciphertext, key []byte) ([]byte, error) {
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}

	iv := ciphertext[:aes.BlockSize]
	data := ciphertext[aes.BlockSize:]
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not aligned to block size")
	}

	plaintext := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, data)

	return pkcs7Unpad(plaintext)
}

// pkcs7Pad PKCS#7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

// pkcs7Unpad PKCS#7 去除填充（含完整性校验）
func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("empty plaintext")
	}
	padLen := int(data[length-1])
	if padLen == 0 || padLen > aes.BlockSize || padLen > length {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := length - padLen; i < length; i++ {
		if data[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}
	return data[:length-padLen], nil
}
