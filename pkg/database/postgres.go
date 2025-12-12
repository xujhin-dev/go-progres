package database

import (
	"fmt"
	"log"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDatabase 初始化数据库连接
func InitDatabase() *gorm.DB {
	cfg := config.GlobalConfig.Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移用户模型
	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
