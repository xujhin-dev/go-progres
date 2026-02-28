package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// SQLCAdapter 将 sqlx.DB 适配为 SQLC 的 DBTX 接口
type SQLCAdapter struct {
	db *sqlx.DB
}

// NewSQLCAdapter 创建新的 SQLC 适配器
func NewSQLCAdapter(db *sqlx.DB) *SQLCAdapter {
	return &SQLCAdapter{db: db}
}

// Exec 实现 SQLC DBTX 接口的 Exec 方法
func (a *SQLCAdapter) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	result, err := a.db.ExecContext(ctx, query, args...)
	if err != nil {
		return pgconn.CommandTag{}, err
	}

	// 将 sql.Result 转换为 pgconn.CommandTag
	rowsAffected, _ := result.RowsAffected()
	return pgconn.NewCommandTag(fmt.Sprintf("ROWS %d", rowsAffected)), nil
}

// Query 实现 SQLC DBTX 接口的 Query 方法
func (a *SQLCAdapter) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	rows, err := a.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return &SQLCRowAdapter{rows: rows, isRow: false}, nil
}

// QueryRow 实现 SQLC DBTX 接口的 QueryRow 方法
func (a *SQLCAdapter) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return &SQLCRowAdapter{row: a.db.QueryRowxContext(ctx, query, args...), isRow: true}
}

// SQLCRowAdapter 简化的适配器实现
type SQLCRowAdapter struct {
	rows  *sqlx.Rows
	row   *sqlx.Row
	isRow bool
}

// Scan 实现 Scan 方法
func (r *SQLCRowAdapter) Scan(dest ...interface{}) error {
	if r.isRow && r.row != nil {
		return r.row.Scan(dest...)
	}
	if !r.isRow && r.rows != nil {
		return r.rows.Scan(dest...)
	}
	return fmt.Errorf("no data available")
}

// Next 实现 Next 方法
func (r *SQLCRowAdapter) Next() bool {
	if !r.isRow && r.rows != nil {
		return r.rows.Next()
	}
	return false
}

// Close 实现 Close 方法
func (r *SQLCRowAdapter) Close() {
	if !r.isRow && r.rows != nil {
		r.rows.Close()
	}
}

// Err 实现 Err 方法
func (r *SQLCRowAdapter) Err() error {
	if !r.isRow && r.rows != nil {
		return r.rows.Err()
	}
	return nil
}

// Values 实现 Values 方法
func (r *SQLCRowAdapter) Values() ([]interface{}, error) {
	if !r.isRow && r.rows != nil {
		return r.rows.SliceScan()
	}
	return nil, fmt.Errorf("not a rows result")
}

// FieldDescriptions 实现 FieldDescriptions 方法
func (r *SQLCRowAdapter) FieldDescriptions() []pgconn.FieldDescription {
	// 简化实现，返回空切片
	return []pgconn.FieldDescription{}
}

// CommandTag 实现 CommandTag 方法
func (r *SQLCRowAdapter) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

// Conn 实现 Conn 方法
func (r *SQLCRowAdapter) Conn() *pgx.Conn {
	// 简化实现，返回 nil
	return nil
}

// RawValues 实现 RawValues 方法
func (r *SQLCRowAdapter) RawValues() [][]byte {
	// 简化实现，返回 nil
	return nil
}

// ScanRow 实现 ScanRow 方法
func (r *SQLCRowAdapter) ScanRow(row []any) error {
	// 简化实现，使用 Scan
	return r.Scan(row...)
}
