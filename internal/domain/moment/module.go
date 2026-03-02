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
	// 1. 依赖注入 - 暂时使用简化仓库进行测试
	mRepo := repository.NewSimpleMomentRepository(ctx.DB)
	momentService := service.NewMomentService(mRepo)
	momentHandler := handler.NewMomentHandler(momentService)

	// 搜索服务
	searchService := service.NewSearchService(mRepo)
	searchHandler := handler.NewSearchHandler(searchService)

	// 2. 路由注册
	setupRoutes(ctx.Router, momentHandler, searchHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.MomentHandler, searchHandler *handler.SearchHandler) {
	// 受保护的路由
	momentGroup := r.Group("/moments")
	momentGroup.Use(middleware.AuthMiddleware())
	{
		momentGroup.POST("/publish", h.PublishPost)
		momentGroup.PUT("/:id/audit", h.AuditPost)
		momentGroup.GET("/feed", h.GetFeed)
		momentGroup.POST("/:id/comments", h.AddComment)
		momentGroup.GET("/:id/comments", h.GetComments)
		momentGroup.POST("/like", h.ToggleLike)
		momentGroup.GET("/topics", h.GetTopics)
		momentGroup.DELETE("/topics/:id", h.DeleteTopic)
	}

	// 搜索路由（部分需要认证）
	searchGroup := r.Group("/search")
	{
		searchGroup.POST("/moments", middleware.AuthMiddleware(), searchHandler.SearchMoments)
		searchGroup.GET("/topics", searchHandler.SearchTopics)
		searchGroup.GET("/hot-topics", searchHandler.GetHotTopics)
		searchGroup.GET("/users/:userId/moments", searchHandler.GetUserMoments)
	}
}
