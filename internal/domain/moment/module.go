package moment

import (
	"user_crud_jwt/internal/domain/moment/handler"
	"user_crud_jwt/internal/domain/moment/repository"
	"user_crud_jwt/internal/domain/moment/service"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"

	"github.com/gin-gonic/gin"
)

// MomentModule 时刻模块
type MomentModule struct{}

func init() {
	registry.Register(&MomentModule{})
}

func (m *MomentModule) Name() string {
	return "moment"
}

func (m *MomentModule) Priority() int {
	return 30
}

func (m *MomentModule) Init(ctx *registry.ModuleContext) error {
	// 1. 依赖注入 - 使用 SQLX 仓库
	mRepo := repository.NewMomentRepository(ctx.DB)
	momentService := service.NewMomentService(mRepo)
	momentHandler := handler.NewMomentHandler(momentService)

	// 2. 路由注册
	setupRoutes(ctx.Router, momentHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.MomentHandler) {
	// 受保护的路由
	momentGroup := r.Group("/moments")
	momentGroup.Use(middleware.AuthMiddleware())
	{
		momentGroup.POST("/publish", h.PublishPost)
		momentGroup.PUT("/:id/audit", h.AuditPost)
	}
}
