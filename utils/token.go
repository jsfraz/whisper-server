package utils

import (
	"fmt"
	"strings"
	"time"

	"crypto/rsa"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
)

// Creates signed access token.
//
//	@param id
//	@return string
//	@return error
func GenerateToken(id string, lifespan int, secret string, tokenId *string) (string, error) {
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

// Returns the user ID (UUID) from a token's subject claim.
func userIdFromClaims(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}
	userId, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("invalid subject claim")
	}
	return userId, nil
}

// Checks if given token is valid.
//
//	@param tokenStr
//	@param secret
//	@return string
//	@return error
func TokenValid(tokenStr string, secret string) (string, *string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", nil, err
	}
	// User ID
	userId, err := userIdFromClaims(token)
	if err != nil {
		return "", nil, err
	}
	tokenId := token.Claims.(jwt.MapClaims)["tokenId"]
	var tokenIdStr *string
	if tokenId != nil {
		str, ok := tokenId.(string)
		if !ok {
			return "", nil, fmt.Errorf("invalid tokenId claim")
		}
		tokenIdStr = &str
	}
	return userId, tokenIdStr, nil
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
// @return string
// @return error
func GetUserIdFromToken(tokenStr string) (string, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return "", err
	}
	return userIdFromClaims(token)
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
