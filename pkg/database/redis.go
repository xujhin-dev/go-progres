package database

import (
	"context"
	"log"
	"time"
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
		// 连接池配置
		PoolSize:     50,              // 连接池大小
		MinIdleConns: 10,              // 最小空闲连接数
		MaxRetries:   3,               // 最大重试次数
		DialTimeout:  time.Second * 5, // 连接超时
		ReadTimeout:  time.Second * 3, // 读取超时
		WriteTimeout: time.Second * 3, // 写入超时
		PoolTimeout:  time.Second * 4, // 连接池超时
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connection established with optimized settings")
	return rdb
}
