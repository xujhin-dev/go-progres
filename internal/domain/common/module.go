package common

import (
	commonHandler "user_crud_jwt/internal/pkg/common"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"

	"github.com/gin-gonic/gin"
)

// CommonModule 通用功能模块
type CommonModule struct{}

func init() {
	registry.Register(&CommonModule{})
}

func (m *CommonModule) Name() string {
	return "common"
}

func (m *CommonModule) Priority() int {
	return 100 // 最后初始化
}

func (m *CommonModule) Init(ctx *registry.ModuleContext) error {
	// 注册通用路由
	setupRoutes(ctx.Router)
	return nil
}

func setupRoutes(r *gin.Engine) {
	// 文件上传接口
	r.POST("/upload", middleware.AuthMiddleware(), commonHandler.UploadFile)
}
