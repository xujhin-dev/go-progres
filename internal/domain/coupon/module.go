package coupon

import (
	"user_crud_jwt/internal/domain/coupon/handler"
	"user_crud_jwt/internal/domain/coupon/repository"
	"user_crud_jwt/internal/domain/coupon/service"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"

	"github.com/gin-gonic/gin"
)

// CouponModule 优惠券模块
type CouponModule struct{}

func init() {
	registry.Register(&CouponModule{})
}

func (m *CouponModule) Name() string {
	return "coupon"
}

func (m *CouponModule) Priority() int {
	return 10
}

func (m *CouponModule) Init(ctx *registry.ModuleContext) error {
	// 1. 依赖注入 - 使用 SQLX 仓库
	cRepo := repository.NewCouponRepository(ctx.DB)
	couponService := service.NewCouponService(cRepo, ctx.Redis)
	couponHandler := handler.NewCouponHandler(couponService)

	// 2. 路由注册
	setupRoutes(ctx.Router, couponHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.CouponHandler) {
	// 公开路由
	couponGroup := r.Group("/coupons")
	{
		couponGroup.POST("/", h.CreateCoupon)
	}

	// 受保护的路由
	protectedGroup := r.Group("/coupons")
	protectedGroup.Use(middleware.AuthMiddleware())
	{
		protectedGroup.POST("/:id/claim", h.ClaimCoupon)
	}
}
