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

// AESEncryptBase64 AES加密转Base64字符串
func AESEncryptBase64(text, key string) (string, error) {
	if text == "" {
		return "", nil
	}
	xpass, err := AESEncrypt([]byte(text), []byte(key))
	if err != nil {
		return "", err
	}
	pass64 := base64.StdEncoding.EncodeToString(xpass)
	return pass64, nil
}

// AESDecryptBase64 AES解密解Base64字符串
func AESDecryptBase64(text, key string) (string, error) {
	if text == "" {
		return "", nil
	}
	bytesPass, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", fmt.Errorf("base64 decode error: %w", err)
	}
	tpass, err := AESDecrypt(bytesPass, []byte(key))
	if err != nil {
		return "", err
	}
	return string(tpass), nil
}

// AESEncrypt AES加密 (CBC模式 + PKCS7填充)
// 密文结构: IV (16 bytes) + Ciphertext
func AESEncrypt(plaintext, aeskey []byte) ([]byte, error) {
	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, fmt.Errorf("aes key error: %w", err)
	}

	// PKCS7 填充
	plaintext = pkcs7Padding(plaintext, aes.BlockSize)

	// 准备密文空间: IV + Plaintext
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("generate iv error: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// AESDecrypt AES解密 (CBC模式 + PKCS7填充)
func AESDecrypt(ciphertext, aeskey []byte) ([]byte, error) {
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, fmt.Errorf("aes key error: %w", err)
	}

	iv := ciphertext[:aes.BlockSize]
	data := ciphertext[aes.BlockSize:]

	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	// 创建副本避免修改原始数据
	plaintext := make([]byte, len(data))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, data)

	// 去除填充
	res, err := pkcs7UnPadding(plaintext)
	if err != nil {
		return nil, fmt.Errorf("unpadding error: %w", err)
	}

	return res, nil
}

// pkcs7Padding PKCS7 填充
func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// pkcs7UnPadding PKCS7 去除填充
func pkcs7UnPadding(plaintext []byte) ([]byte, error) {
	length := len(plaintext)
	if length == 0 {
		return nil, fmt.Errorf("plaintext is empty")
	}
	unpadding := int(plaintext[length-1])
	if unpadding > length || unpadding == 0 {
		return nil, fmt.Errorf("invalid padding size")
	}
	// 校验所有填充字节是否一致
	for i := length - unpadding; i < length; i++ {
		if int(plaintext[i]) != unpadding {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}
	return plaintext[:(length - unpadding)], nil
}
