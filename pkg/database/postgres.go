package database

import (
	"fmt"
	"log"
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
