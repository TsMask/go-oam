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
	if len(text) == 0 {
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
	if len(text) == 0 {
		return "", nil
	}
	bytesPass, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	tpass, err := AESDecrypt(bytesPass, []byte(key))
	if err != nil {
		return "", err
	}
	return string(tpass), nil
}

// AESEncrypt AES加密
func AESEncrypt(plaintext, aeskey []byte) ([]byte, error) {
	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, err
	}
	blockSize := aes.BlockSize

	padding := blockSize - (len(plaintext) % blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	plaintext = append(plaintext, padtext...)

	ciphertext := make([]byte, blockSize+len(plaintext))
	iv := ciphertext[:blockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[blockSize:], plaintext)

	return ciphertext, nil
}

// AESDecrypt AES解密
func AESDecrypt(ciphertext, aeskey []byte) ([]byte, error) {
	blockSize := aes.BlockSize
	if len(ciphertext) < blockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:blockSize]
	ciphertext = ciphertext[blockSize:]
	block, err := aes.NewCipher([]byte(aeskey))

	if err != nil {
		return nil, err
	}
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is invalid")
	}
	if len(ciphertext)%blockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除填充
	padding := int(ciphertext[len(ciphertext)-1])
	if padding > blockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	ciphertext = ciphertext[:len(ciphertext)-padding]

	return ciphertext, nil
}
