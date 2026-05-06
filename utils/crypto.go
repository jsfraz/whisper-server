package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
)

// Returns cryptographically secure random string.
//
//	@param length
//	@return string
func RandomASCIIString(length int) (string, error) {
	// Chars
	const charset = "!%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	// Byte slice for result
	result := make([]byte, length)
	// Generate random chars
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
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
//
//	@param publicKey
//	@return error
func validateRsaPublicKey(publicKey *rsa.PublicKey) error {
	if publicKey.N.BitLen() < 4096 {
		return errors.New("RSA key size is too small")
	}
	return nil
}
