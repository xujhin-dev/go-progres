package router

import (
	"user_crud_jwt/internal/domain/user/handler"
	"user_crud_jwt/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes 设置用户模块路由
func SetupUserRoutes(r *gin.Engine, userHandler *handler.UserHandler) {
	// 公开路由
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
	}

	// 受保护的路由
	userGroup := r.Group("/users")
	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/", userHandler.GetUsers)
		userGroup.GET("/:id", userHandler.GetUser)
		userGroup.PUT("/:id", userHandler.UpdateUser)
		userGroup.DELETE("/:id", userHandler.DeleteUser)
	}
}
