package utils

import (
	"time"
	"user_crud_jwt/internal/pkg/config"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken 生成JWT Token
func GenerateToken(userID uint, role int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * time.Duration(config.GlobalConfig.JWT.Expire)).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GlobalConfig.JWT.Secret))
}

// ParseToken 验证JWT Token
func ParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(config.GlobalConfig.JWT.Secret), nil
	})
}
