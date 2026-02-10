package middleware

import (
	"net/http"
	"strings"

	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/response"
	"user_crud_jwt/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 检查格式 "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := utils.ParseToken(tokenString)
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, response.ErrTokenInvalid, "Invalid or expired token")
			c.Abort()
			return
		}

		// 将 userID 和 role 存入上下文
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("userID", claims["user_id"])
			c.Set("role", claims["role"])
		}

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Error(c, http.StatusUnauthorized, response.ErrNoPermission, "Unauthorized")
			c.Abort()
			return
		}

		// JSON解析出来的数字可能是 float64
		var roleInt int
		switch v := role.(type) {
		case float64:
			roleInt = int(v)
		case int:
			roleInt = v
		default:
			response.Error(c, http.StatusForbidden, response.ErrNoPermission, "Invalid role format")
			c.Abort()
			return
		}

		if roleInt != model.RoleAdmin {
			response.Error(c, http.StatusForbidden, response.ErrNoPermission, "Admin permission required")
			c.Abort()
			return
		}

		c.Next()
	}
}
