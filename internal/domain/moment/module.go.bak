package moment

import (
	"user_crud_jwt/internal/domain/moment/handler"
	"user_crud_jwt/internal/domain/moment/repository"
	"user_crud_jwt/internal/domain/moment/service"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"

	"github.com/gin-gonic/gin"
)

// MomentModule 动态模块
type MomentModule struct{}

func init() {
	registry.Register(&MomentModule{})
}

func (m *MomentModule) Name() string {
	return "moment"
}

func (m *MomentModule) Priority() int {
	return 10
}

func (m *MomentModule) Init(ctx *registry.ModuleContext) error {
	// 1. 依赖注入
	mRepo := repository.NewMomentRepository(ctx.DB)
	mService := service.NewMomentService(mRepo)
	mHandler := handler.NewMomentHandler(mService)

	// 2. 路由注册
	setupRoutes(ctx.Router, mHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.MomentHandler) {
	g := r.Group("/moments")

	// Public (or semi-public) feed
	g.GET("/feed", h.GetFeed)
	g.GET("/:id/comments", h.GetComments)
	g.GET("/topics", h.GetTopics)

	// User interactions (Requires Login)
	auth := g.Group("")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/publish", h.PublishPost)
		auth.POST("/:id/comment", h.AddComment)
		auth.POST("/like", h.ToggleLike)
	}

	// Admin (Requires Admin Role)
	admin := g.Group("")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		admin.PUT("/:id/audit", h.AuditPost)
		admin.DELETE("/topics/:id", h.DeleteTopic)
	}
}
