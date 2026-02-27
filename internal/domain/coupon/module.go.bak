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
	// 1. 依赖注入
	cRepo := repository.NewCouponRepository(ctx.DB)
	cService := service.NewCouponService(cRepo, ctx.Redis)
	cHandler := handler.NewCouponHandler(cService)

	// 2. 路由注册
	setupRoutes(ctx.Router, cHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.CouponHandler) {
	g := r.Group("/coupons")

	// 需要认证的路由组
	authorized := g.Group("")
	authorized.Use(middleware.AuthMiddleware())
	{
		// 抢券需要登录
		authorized.POST("/:id/claim", h.ClaimCoupon)

		// 需要管理员权限的路由组
		admin := authorized.Group("")
		admin.Use(middleware.AdminMiddleware())
		{
			// 创建优惠券
			admin.POST("/", h.CreateCoupon)
			// 管理员给用户发券
			admin.POST("/send", h.SendCoupon)
		}
	}
}
