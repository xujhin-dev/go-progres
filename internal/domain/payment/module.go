package payment

import (
	"user_crud_jwt/internal/domain/payment/handler"
	"user_crud_jwt/internal/domain/payment/repository"
	"user_crud_jwt/internal/domain/payment/service"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"

	"github.com/gin-gonic/gin"
)

// PaymentModule 支付模块
type PaymentModule struct{}

func init() {
	registry.Register(&PaymentModule{})
}

func (m *PaymentModule) Name() string {
	return "payment"
}

func (m *PaymentModule) Priority() int {
	return 20
}

func (m *PaymentModule) Init(ctx *registry.ModuleContext) error {
	// 1. 依赖注入 - 使用 SQLX 仓库
	pRepo := repository.NewPaymentRepository(ctx.DB)

	// 暂时使用 nil 作为用户服务，后续可以修复
	paymentService := service.NewPaymentService(pRepo, nil)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	// 2. 路由注册
	setupRoutes(ctx.Router, paymentHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.PaymentHandler) {
	// 受保护的路由
	paymentGroup := r.Group("/payments")
	paymentGroup.Use(middleware.AuthMiddleware())
	{
		paymentGroup.POST("/orders", h.CreateOrder)
	}
}
