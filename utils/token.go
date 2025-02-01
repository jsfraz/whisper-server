package utils

import (
	"fmt"
	"strings"
	"time"

	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Creates signed access token.
//
//	@param id
//	@return string
//	@return error
func GenerateToken(id uint64, lifespan int, secret string, tokenId *string) (string, error) {
	// Payload
	now := time.Now()
	claims := jwt.MapClaims{}
	claims["sub"] = id
	claims["exp"] = now.Add(time.Second * time.Duration(lifespan)).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
	if tokenId != nil {
		claims["tokenId"] = *tokenId
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create and sign token
	return token.SignedString([]byte(secret))
}

// Checks if given token is valid.
//
//	@param tokenStr
//	@param secret
//	@return uint64
//	@return error
func TokenValid(tokenStr string, secret string) (uint64, *string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, nil, err
	}
	// User ID
	claims, _ := token.Claims.(jwt.MapClaims)
	userId := claims["sub"].(float64)
	tokenId := claims["tokenId"]
	var tokenIdStr *string
	if tokenId != nil {
		str := tokenId.(string)
		tokenIdStr = &str
	}
	return uint64(userId), tokenIdStr, nil
}

// Gets token from Gin context.
//
// @param c
// @return string
func ExtractTokenFromContext(c *gin.Context) string {
	bearerToken := c.Request.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return ""
}

// Gets user ID from JWT token signed with RSA key.
//
// @param tokenStr
// @return uint64
// @return error
func GetUserIdFromToken(tokenStr string) (uint64, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return 0, err
	}
	// User ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}
	userId := claims["sub"].(float64)
	return uint64(userId), nil
}

// Validates RSA JWT token.
//
// @param tokenStr
// @param publicKey
// @return error
func ValidateRsaJwtToken(tokenStr string, publicKey *rsa.PublicKey) error {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	return nil
}
