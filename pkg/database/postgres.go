package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"user_crud_jwt/internal/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDatabase 初始化数据库连接
func InitDatabase() *gorm.DB {
	cfg := config.GlobalConfig.Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info), // 生产环境可改为 Warn
		PrepareStmt:                              true,                                // 预编译 SQL 缓存
		DisableForeignKeyConstraintWhenMigrating: true,                                // 禁用外键约束检查
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 获取底层 SQL DB 对象以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// 连接池配置
	configureConnectionPool(sqlDB)

	// 生产环境建议使用 golang-migrate，这里保留 AutoMigrate 仅作演示或开发环境使用
	// err = db.AutoMigrate(
	// 	&userModel.User{},
	// 	&couponModel.Coupon{},
	// 	&couponModel.UserCoupon{},
	// )
	// if err != nil {
	// 	log.Fatalf("Failed to migrate database: %v", err)
	// }

	return db
}

// configureConnectionPool 配置数据库连接池
func configureConnectionPool(sqlDB *sql.DB) {
	// 设置连接池中的最大连接数
	sqlDB.SetMaxOpenConns(100) // 根据数据库服务器性能调整

	// 设置连接池中的最大空闲连接数
	sqlDB.SetMaxIdleConns(10) // 推荐 SetMaxOpenConns 的 10%

	// 设置连接的最大生命周期
	sqlDB.SetConnMaxLifetime(time.Hour) // 1小时，避免长时间连接问题

	// 设置连接的最大空闲时间
	sqlDB.SetConnMaxIdleTime(time.Minute * 30) // 30分钟

	log.Println("Database connection pool configured successfully")
}
