package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"user_crud_jwt/internal/pkg/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// DB wraps sqlx.DB for additional functionality
type DB struct {
	*sqlx.DB
}

// InitDatabase 初始化数据库连接
func InitDatabase() *DB {
	cfg := config.GlobalConfig.Database

	// Build connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)

	// Connect using pgx driver
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool
	configureConnectionPool(db.DB)

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connected successfully")
	return &DB{DB: db}
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

// BeginTx 开始事务
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return db.DB.BeginTxx(ctx, opts)
}

// ExecContext 执行SQL语句
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.DB.ExecContext(ctx, query, args...)
}

// QueryContext 查询多行
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	return db.DB.QueryxContext(ctx, query, args...)
}

// QueryRowContext 查询单行
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return db.DB.QueryRowxContext(ctx, query, args...)
}

// GetContext 查询单行到结构体
func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return db.DB.GetContext(ctx, dest, query, args...)
}

// SelectContext 查询多行到切片
func (db *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return db.DB.SelectContext(ctx, dest, query, args...)
}

// NamedExec 执行命名参数SQL
func (db *DB) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return db.DB.NamedExecContext(ctx, query, arg)
}

// NamedQuery 查询命名参数SQL
func (db *DB) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	return db.DB.NamedQueryContext(ctx, query, arg)
}

// Reconnect 重新连接数据库
func (db *DB) Reconnect() error {
	if err := db.DB.Ping(); err != nil {
		log.Printf("Database connection lost, attempting to reconnect: %v", err)

		// Close existing connection
		db.DB.Close()

		// Reconnect
		cfg := config.GlobalConfig.Database
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)

		newDB, err := sqlx.Connect("postgres", dsn)
		if err != nil {
			return fmt.Errorf("failed to reconnect to database: %v", err)
		}

		// Update the underlying DB
		db.DB = newDB
		configureConnectionPool(newDB.DB)

		log.Println("Database reconnected successfully")
	}
	return nil
}
