package utils

import (
	"fmt"
	"strings"
	"time"

	"crypto/rsa"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
)

// Token types. A token is only accepted for the purpose it was issued for, so
// three different secrets are no longer the only thing keeping them apart.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	TokenTypeWs      = "ws"
)

// TokenClaims is what a verified token says. DeviceId is the device the token
// was issued to: a revoked device keeps its tokens, they just stop working.
type TokenClaims struct {
	UserId   string
	DeviceId string
	TokenId  *string
}

// Creates signed access token.
//
//	@param id
//	@param deviceId
//	@param lifespan
//	@param secret
//	@param tokenId
//	@param tokenType
//	@return string
//	@return error
func GenerateToken(id string, deviceId string, lifespan int, secret string, tokenId *string, tokenType string) (string, error) {
	// Payload
	now := time.Now()
	claims := jwt.MapClaims{}
	claims["sub"] = id
	claims["dev"] = deviceId
	claims["typ"] = tokenType
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

// Checks if given token is valid and was issued for [expectedType].
//
//	@param tokenStr
//	@param secret
//	@param expectedType
//	@return *TokenClaims
//	@return error
func TokenValid(tokenStr string, secret string, expectedType string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	// User ID
	userId, err := userIdFromClaims(token)
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(jwt.MapClaims)
	// A token minted for another purpose (or before types existed) is refused.
	if claims["typ"] != expectedType {
		return nil, fmt.Errorf("unexpected token type")
	}
	// Every token belongs to a device. A token without one predates the device
	// registry and cannot be checked for revocation, so it is refused.
	deviceId, ok := claims["dev"].(string)
	if !ok || deviceId == "" {
		return nil, fmt.Errorf("invalid device claim")
	}
	tokenId := claims["tokenId"]
	var tokenIdStr *string
	if tokenId != nil {
		str, ok := tokenId.(string)
		if !ok {
			return nil, fmt.Errorf("invalid tokenId claim")
		}
		tokenIdStr = &str
	}
	return &TokenClaims{UserId: userId, DeviceId: deviceId, TokenId: tokenIdStr}, nil
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

// Returns a string claim from an unverified token. Callers must verify the
// signature first; the value is only trusted once they have.
//
//	@param tokenStr
//	@param claim
//	@return string
//	@return error
func GetStringClaimFromToken(tokenStr string, claim string) (string, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}
	value, ok := claims[claim].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("missing %s claim", claim)
	}
	return value, nil
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
