package utils

import (
	"time"
	"user_crud_jwt/internal/pkg/config"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义JWT Claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   int    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token
func GenerateToken(userID string, role int) (string, error) {
	now := time.Now()
	expireTime := now.Add(time.Duration(config.GlobalConfig.JWT.Expire) * time.Hour)

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			Issuer:    "user-crud",
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tokenClaims.SignedString([]byte(config.GlobalConfig.JWT.Secret))
}

// ParseToken 验证JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(config.GlobalConfig.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}
