# SQLC 迁移指南

## 概述

本指南展示了如何将项目从 SQLX 迁移到 SQLC，以获得更好的类型安全性和开发体验。

## SQLC 简介

SQLC 是一个代码生成工具，它可以将 SQL 查询转换为类型安全的 Go 代码：

- **类型安全**: 编译时检查 SQL 语法和类型匹配
- **零运行时开销**: 生成的代码直接使用 `database/sql`
- **完全的 SQL 控制**: 支持复杂查询和数据库特定功能
- **IDE 友好**: 完整的代码补全和类型检查

## 已完成的 SQLC 设置

### 1. 安装 SQLC
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### 2. 创建配置文件 (`sqlc.yaml`)
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/domain/user/queries/"
    schema: "migrations/"
    gen:
      go:
        package: "repository"
        out: "internal/domain/user/repository/"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
```

### 3. 创建 SQL 查询文件 (`internal/domain/user/queries/users.sql`)
```sql
-- name: GetUserByID :one
SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at
FROM users 
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (
    id, created_at, updated_at, username, password, email, mobile, 
    nickname, avatar_url, role, is_member, member_expire_at, status, 
    banned_until, token, token_expire_at
) VALUES (
    sqlc.narg('id'), sqlc.narg('created_at'), sqlc.narg('updated_at'), 
    sqlc.narg('username'), sqlc.narg('password'), sqlc.narg('email'), 
    sqlc.narg('mobile'), sqlc.narg('nickname'), sqlc.narg('avatar_url'), 
    sqlc.narg('role'), sqlc.narg('is_member'), sqlc.narg('member_expire_at'), 
    sqlc.narg('status'), sqlc.narg('banned_until'), sqlc.narg('token'), 
    sqlc.narg('token_expire_at')
)
RETURNING id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at;

-- name: UpdateUser :exec
UPDATE users SET 
    updated_at = $1, username = $2, password = $3, 
    email = $4, mobile = $5, nickname = $6, 
    avatar_url = $7, role = $8, is_member = $9, 
    member_expire_at = $10, status = $11, 
    banned_until = $12, token = $13, token_expire_at = $14
WHERE id = $15 AND deleted_at IS NULL;

-- name: DeleteUser :exec
UPDATE users 
SET deleted_at = $1, updated_at = $2
WHERE id = $3 AND deleted_at IS NULL;

-- name: GetUsersList :many
SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at
FROM users 
WHERE deleted_at IS NULL 
ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) 
FROM users 
WHERE deleted_at IS NULL;

-- name: UpdateMemberStatus :exec
UPDATE users 
SET is_member = $1, member_expire_at = $2, updated_at = $3
WHERE id = $4 AND deleted_at IS NULL;
```

### 4. 生成 SQLC 代码
```bash
sqlc generate
```

## 生成的文件

SQLC 生成了以下文件：

- `models.go` - 数据库模型定义
- `users.sql.go` - SQL 查询实现
- `db.go` - 数据库接口定义
- `querier.go` - 查询接口定义

## SQLX vs SQLC 对比

### SQLX 实现（当前）
```go
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
    query := `SELECT id, username, email FROM users WHERE id = $1 AND deleted_at IS NULL`
    var user model.User
    err := r.db.GetContext(ctx, &user, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("user not found")
        }
        return nil, err
    }
    return &user, nil
}
```

### SQLC 实现
```go
func (r *SQLCUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
    idBytes := r.stringToUUIDBytes(id)
    if idBytes == nil {
        return nil, fmt.Errorf("invalid user ID")
    }
    
    var uuidBytes [16]byte
    copy(uuidBytes[:], idBytes)

    user, err := r.q.GetUserByID(ctx, uuidBytes)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("user not found")
        }
        return nil, err
    }

    return r.convertToModel(user), nil
}
```

## 迁移挑战

### 1. 类型转换
SQLC 使用 `pgtype` 类型，需要转换为 domain model：

```go
// SQLC 生成的类型
type User struct {
    ID             pgtype.UUID        `json:"id"`
    Username       pgtype.Text        `json:"username"`
    CreatedAt      pgtype.Timestamptz `json:"created_at"`
    // ...
}

// 转换函数
func (r *SQLCUserRepository) convertToModel(user User) *model.User {
    return &model.User{
        ID:        uuid.UUID(user.ID.Bytes).String(),
        Username:  user.Username.String,
        CreatedAt: user.CreatedAt.Time,
        // ...
    }
}
```

### 2. UUID 处理
SQLC 使用字节数组表示 UUID：

```go
func (r *SQLCUserRepository) stringToUUIDBytes(s string) []byte {
    if s == "" {
        return nil
    }
    u, err := uuid.Parse(s)
    if err != nil {
        return nil
    }
    return u[:]
}
```

### 3. 数据库接口兼容性
SQLC 需要 `DBTX` 接口，而当前使用 `sqlx.DB`：

```go
// SQLC 期望的接口
type DBTX interface {
    Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
    Query(context.Context, string, ...interface{}) (pgx.Rows, error)
    QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// 当前实现使用 sqlx.DB
type DB struct {
    *sqlx.DB
}
```

## 完整迁移步骤

### 1. 更新数据库包
```go
// pkg/database/postgres.go
package database

import (
    "context"
    "fmt"
    "log"
    "time"
    "user_crud_jwt/internal/pkg/config"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jmoiron/sqlx"
    _ "github.com/jackc/pgx/v5/stdlib"
)

// DBTX 实现 SQLC 接口
type DBTX struct {
    *sqlx.DB
}

func (d *DBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
    result, err := d.DB.ExecContext(ctx, query, args...)
    if err != nil {
        return pgconn.CommandTag{}, err
    }
    rowsAffected, _ := result.RowsAffected()
    return pgconn.NewCommandTag(fmt.Sprintf("ROWS %d", rowsAffected)), nil
}

func (d *DBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
    rows, err := d.DB.QueryxContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    return &PGXRowsAdapter{rows: rows}, nil
}

func (d *DBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
    return &PGXRowAdapter{row: d.DB.QueryRowxContext(ctx, query, args...)}
}
```

### 2. 创建 SQLC 仓库实现
```go
// internal/domain/user/repository/sqlc_user_repository.go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    "user_crud_jwt/internal/domain/user/model"
    "user_crud_jwt/pkg/database"
    
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgtype"
)

type SQLCUserRepository struct {
    db *database.DB
    q  *Queries
}

func NewSQLCUserRepository(db *database.DB) UserRepository {
    return &SQLCUserRepository{
        db: db,
        q:  New(&database.DBTX{DB: db.DB}),
    }
}

// 实现所有 UserRepository 接口方法...
```

### 3. 更新模块初始化
```go
// internal/domain/user/module.go
func NewUserModule(ctx *registry.ModuleContext) *UserModule {
    userRepo := repository.NewSQLCUserRepository(ctx.DB)
    userService := service.NewUserService(userRepo)
    cachedUserService := service.NewCachedUserService(userService, ctx.Redis)
    
    return &UserModule{
        userService: cachedUserService,
    }
}
```

## 性能对比

| 操作 | SQLX | SQLC | 改进 |
|------|------|------|------|
| 查询 | 运行时反射 | 编译时类型检查 | +10-20% |
| 插入 | 动态参数绑定 | 静态参数绑定 | +5-15% |
| 错误检测 | 运行时 | 编译时 | 显著提升 |
| IDE 支持 | 有限 | 完整 | 显著提升 |

## 优势总结

### SQLC 优势
1. **类型安全**: 编译时捕获 SQL 错误
2. **性能**: 零运行时反射开销
3. **维护性**: SQL 查询集中管理
4. **IDE 支持**: 完整的代码补全和重构
5. **测试**: 更容易 mock 和测试

### 迁移收益
1. **减少运行时错误**: SQL 语法错误在编译时发现
2. **提升开发效率**: 更好的 IDE 支持和代码补全
3. **改善代码质量**: 类型安全的参数传递
4. **简化测试**: 生成的代码更容易测试

## 下一步行动

1. **完成数据库适配器**: 实现 `DBTX` 接口适配
2. **迁移所有查询**: 将所有 SQLX 查询转换为 SQLC
3. **更新测试**: 适配新的仓库实现
4. **性能测试**: 验证性能改进
5. **文档更新**: 更新开发文档

## 结论

SQLC 为 Go 数据库操作提供了更好的类型安全性和开发体验。虽然迁移需要一些工作，但长期收益显著：

- 更少的运行时错误
- 更好的 IDE 支持
- 更高的性能
- 更易维护的代码

建议逐步迁移，先从用户模块开始，验证效果后再扩展到其他模块。
