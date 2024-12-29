package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Creates signed access token.
//
//	@param id
//	@return string
//	@return error
func GenerateToken(id uint64, lifespan int, secret string) (string, error) {
	// Payload
	now := time.Now()
	claims := jwt.MapClaims{}
	claims["sub"] = id
	claims["exp"] = now.Add(time.Second * time.Duration(lifespan)).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
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
func TokenValid(tokenStr string, secret string) (uint64, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}
	// User ID
	claims, _ := token.Claims.(jwt.MapClaims)
	fId := claims["sub"].(float64)
	return uint64(fId), nil
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
