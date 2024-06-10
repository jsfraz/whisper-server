package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"math/rand"
	"time"
)

// https://www.calhoun.io/creating-random-strings-in-go/
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// https://gist.github.com/yingray/57fdc3264b1927ef0f984b533d63abab
func Aes256Encrypt(input []byte, key []byte, iv []byte) ([]byte, error) {
	bPlaintext := pkcs5Padding(input, 16, len(input))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(bPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, bPlaintext)
	return ciphertext, nil
}

func pkcs5Padding(ciphertext []byte, blockSize int, after int) []byte {
	padding := (blockSize - len(ciphertext)%blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
