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
func GenerateToken(userID string, role int) (string, *time.Time, error) {
	now := time.Now()
	// 设置token过期时间为1个月
	expireTime := now.Add(30 * 24 * time.Hour) // 30天

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			Issuer:    "user-crud",
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString([]byte(config.GlobalConfig.JWT.Secret))
	if err != nil {
		return "", nil, err
	}
	return token, &expireTime, nil
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
