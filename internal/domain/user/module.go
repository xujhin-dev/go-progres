package user

import (
	"user_crud_jwt/internal/domain/user/handler"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/internal/domain/user/service"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"

	"github.com/gin-gonic/gin"
)

// UserModule 用户模块
type UserModule struct{}

func init() {
	// 自动注册模块
	registry.Register(&UserModule{})
}

func (m *UserModule) Name() string {
	return "user"
}

func (m *UserModule) Priority() int {
	// 用户模块优先级最高，因为其他模块可能依赖它
	return 1
}

func (m *UserModule) Init(ctx *registry.ModuleContext) error {
	// 1. 依赖注入
	userRepo := repository.NewUserRepository(ctx.DB)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	// 2. 路由注册
	setupRoutes(ctx.Router, userHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.UserHandler) {
	// 公开路由
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
	}

	// 受保护的路由
	userGroup := r.Group("/users")
	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/", h.GetUsers)
		userGroup.GET("/:id", h.GetUser)
		userGroup.PUT("/:id", h.UpdateUser)
		userGroup.DELETE("/:id", h.DeleteUser)
		userGroup.PUT("/password", h.ChangePassword)
	}
}
