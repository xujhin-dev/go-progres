package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user_crud_jwt/internal/pkg/config"
	"user_crud_jwt/internal/pkg/registry"
	"user_crud_jwt/pkg/database"

	// 导入所有域模块以触发 init() 函数
	_ "user_crud_jwt/internal/domain/common"
	_ "user_crud_jwt/internal/domain/coupon"
	_ "user_crud_jwt/internal/domain/moment"
	_ "user_crud_jwt/internal/domain/payment"
	_ "user_crud_jwt/internal/domain/user"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	config.LoadConfig()
	cfg := config.GlobalConfig

	// 2. 初始化数据库
	db := database.InitDatabase()
	defer db.DB.Close()

	// 2.5. 初始化 Redis
	redis := database.InitRedis()
	defer redis.Close()

	// 3. 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 4. 创建路由
	router := gin.Default()

	// 5. 初始化模块系统
	moduleCtx := &registry.ModuleContext{
		DB:     db,
		Redis:  redis,
		Router: router,
	}

	if err := registry.InitModules(moduleCtx); err != nil {
		log.Fatalf("Failed to initialize modules: %v", err)
	}

	// 6. 启动服务器
	go func() {
		addr := ":" + cfg.Server.Port
		log.Printf("Starting server on %s", addr)
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 7. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 这里可以添加数据库连接池关闭等清理工作
	_ = ctx

	log.Println("Server exited")
}
