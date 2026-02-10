package payment

import (
	"user_crud_jwt/internal/domain/payment/handler"
	"user_crud_jwt/internal/domain/payment/repository"
	"user_crud_jwt/internal/domain/payment/service"
	"user_crud_jwt/internal/domain/payment/strategy"
	userService "user_crud_jwt/internal/domain/user/service"
	userRepo "user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/internal/pkg/config"
	"user_crud_jwt/internal/pkg/middleware"
	"user_crud_jwt/internal/pkg/registry"
	"user_crud_jwt/pkg/logger"

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
	// 支付模块依赖用户模块，所以优先级较低
	return 20
}

func (m *PaymentModule) Init(ctx *registry.ModuleContext) error {
	// 1. 依赖注入
	pRepo := repository.NewPaymentRepository(ctx.DB)

	// 支付模块依赖用户服务
	uRepo := userRepo.NewUserRepository(ctx.DB)
	uService := userService.NewUserService(uRepo)

	pService := service.NewPaymentService(pRepo, uService)

	// 2. 注册支付策略
	// 支付宝
	if config.GlobalConfig.Alipay.AppID != "" {
		alipayStrategy, err := strategy.NewAlipayStrategy()
		if err != nil {
			logger.Log.Error("Failed to init Alipay strategy: " + err.Error())
		} else {
			pService.RegisterStrategy("alipay", alipayStrategy)
		}
	}

	// 微信支付
	if config.GlobalConfig.Wechat.MchID != "" {
		wechatStrategy, err := strategy.NewWechatStrategy()
		if err != nil {
			logger.Log.Error("Failed to init Wechat strategy: " + err.Error())
		} else {
			pService.RegisterStrategy("wechat", wechatStrategy)
		}
	}

	pHandler := handler.NewPaymentHandler(pService)

	// 3. 路由注册
	setupRoutes(ctx.Router, pHandler)

	return nil
}

func setupRoutes(r *gin.Engine, h *handler.PaymentHandler) {
	g := r.Group("/payment")

	// 支付回调 (无需鉴权，但需验签)
	g.POST("/notify/alipay", h.AlipayNotify)
	g.POST("/notify/wechat", h.WechatNotify)

	// 需要鉴权的接口
	auth := g.Group("")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/order", h.CreateOrder)
	}
}
