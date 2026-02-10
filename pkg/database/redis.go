package database

import (
	"context"
	"log"
	"user_crud_jwt/internal/pkg/config"

	"github.com/redis/go-redis/v9"
)

// InitRedis 初始化 Redis 连接
func InitRedis() *redis.Client {
	cfg := config.GlobalConfig.Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return rdb
}
