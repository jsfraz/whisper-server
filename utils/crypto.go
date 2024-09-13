package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/rand"
	"time"
)

// Reurns random string.
//
//	@param length
//	@return string
func RandomASCIIString(length int) string {
	// Chars
	const charset = "!#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	// Random generator
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	// Byte slice for result
	result := make([]byte, length)
	// Generate random chars
	for i := range result {
		result[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(result)
}

// Load and validate RSA public key.
//
//	@param pemData
//	@return *rsa.PublicKey
//	@return error
func LoadRsaPublicKey(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid PEM block")
	}
	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := parsedKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	// Validate
	err = validateRsaPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}

// Validate RSA public key.
func validateRsaPublicKey(publicKey *rsa.PublicKey) error {
	if publicKey.N.BitLen() < 4096 {
		return errors.New("RSA key size is too small")
	}
	return nil
}

/*
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
	bPlaintext := pkcs5Padding(input, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(bPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, bPlaintext)
	return ciphertext, nil
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := (blockSize - len(ciphertext)%blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
*/
