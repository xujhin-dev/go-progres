package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"user_crud_jwt/internal/pkg/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UnifiedDB 统一的数据库接口，支持SQLC和原生操作
type UnifiedDB struct {
	pool *pgxpool.Pool
}

// InitUnifiedDatabase 初始化统一数据库连接
func InitUnifiedDatabase() *UnifiedDB {
	cfg := config.GlobalConfig.Database

	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)

	// 配置pgx连接池
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Failed to parse database config: %v", err)
	}

	// 优化连接池配置
	poolConfig.MaxConns = 100
	poolConfig.MinConns = 10
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}

	// 测试连接
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Unified database connected successfully with pgxpool")
	return &UnifiedDB{pool: pool}
}

// GetPool 获取pgx连接池
func (db *UnifiedDB) GetPool() *pgxpool.Pool {
	return db.pool
}

// Close 关闭数据库连接
func (db *UnifiedDB) Close() {
	if db.pool != nil {
		db.pool.Close()
		log.Println("Database connection pool closed")
	}
}

// BeginTx 开始事务
func (db *UnifiedDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// Exec 执行SQL语句
func (db *UnifiedDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, query, args...)
}

// Query 查询多行
func (db *UnifiedDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, query, args...)
}

// QueryRow 查询单行
func (db *UnifiedDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, query, args...)
}

// GetDBTX 获取SQLC兼容的DBTX接口
func (db *UnifiedDB) GetDBTX() *pgxpool.Pool {
	return db.pool
}
