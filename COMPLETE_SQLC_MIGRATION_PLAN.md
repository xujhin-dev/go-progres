# 🎯 完整 SQLC 迁移计划

## 📊 当前状态

| 模块 | 当前实现 | SQLC 准备状态 | 迁移优先级 |
|------|----------|---------------|------------|
| **用户模块** | SQLX | ✅ 完整 | 🟢 高 |
| **优惠券模块** | GORM | ✅ 完整 | 🟡 中 |
| **支付模块** | GORM | ✅ 完整 | 🟡 中 |
| **时刻模块** | GORM | ✅ 完整 | 🟡 中 |

## 🚀 迁移策略

### 阶段 1: 用户模块 SQLC 迁移 (高优先级)

#### 1.1 创建 SQLC 用户仓库
```go
// internal/domain/user/repository/user_repository_sqlc.go
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
    q  *Queries  // SQLC 生成的查询
}

func NewSQLCUserRepository(db *database.DB) UserRepository {
    // 创建适配器
    adapter := NewSQLCAdapter(db.DB)
    return &SQLCUserRepository{
        db: db,
        q:  New(adapter),
    }
}

// 实现所有 UserRepository 接口方法...
```

#### 1.2 类型转换函数
```go
// SQLC User -> Domain User
func (r *SQLCUserRepository) convertToModel(user User) *model.User {
    return &model.User{
        ID:        uuid.UUID(user.ID.Bytes).String(),
        Username:  user.Username.String,
        CreatedAt: user.CreatedAt.Time,
        // ... 其他字段转换
    }
}

// Domain User -> SQLC User
func (r *SQLCUserRepository) convertToSQLC(user *model.User) User {
    idBytes := r.stringToUUIDBytes(user.ID)
    var uuidBytes [16]byte
    copy(uuidBytes[:], idBytes)
    
    return User{
        ID:        pgtype.UUID{Bytes: uuidBytes, Valid: user.ID != ""},
        Username:  pgtype.Text{String: user.Username, Valid: true},
        // ... 其他字段转换
    }
}
```

#### 1.3 更新模块初始化
```go
// internal/domain/user/module.go
func NewUserModule(ctx *registry.ModuleContext) *UserModule {
    // 使用 SQLC 仓库
    userRepo := repository.NewSQLCUserRepository(ctx.DB)
    userService := service.NewUserService(userRepo)
    cachedUserService := service.NewCachedUserService(userService, ctx.Redis)
    
    return &UserModule{
        userService: cachedUserService,
    }
}
```

### 阶段 2: 其他模块 SQLC 迁移 (中优先级)

#### 2.1 优惠券模块迁移
```go
// internal/domain/coupon/repository/coupon_repository_sqlc.go
type SQLCCouponRepository struct {
    db *database.DB
    q  *Queries
}

func NewSQLCCouponRepository(db *database.DB) CouponRepository {
    adapter := NewSQLCAdapter(db.DB)
    return &SQLCCouponRepository{
        db: db,
        q:  New(adapter),
    }
}
```

#### 2.2 支付模块迁移
```go
// internal/domain/payment/repository/payment_repository_sqlc.go
type SQLCPaymentRepository struct {
    db *database.DB
    q  *Queries
}

func NewSQLCPaymentRepository(db *database.DB) PaymentRepository {
    adapter := NewSQLCAdapter(db.DB)
    return &SQLCPaymentRepository{
        db: db,
        q:  New(adapter),
    }
}
```

#### 2.3 时刻模块迁移
```go
// internal/domain/moment/repository/moment_repository_sqlc.go
type SQLCMomentRepository struct {
    db *database.DB
    q  *Queries
}

func NewSQLCMomentRepository(db *database.DB) MomentRepository {
    adapter := NewSQLCAdapter(db.DB)
    return &SQLCMomentRepository{
        db: db,
        q:  New(adapter),
    }
}
```

### 阶段 3: 数据库适配器完善

#### 3.1 完整的 SQLC 适配器
```go
// pkg/database/sqlc_adapter.go
package database

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgproto3"
    "github.com/jmoiron/sqlx"
)

type SQLCAdapter struct {
    db *sqlx.DB
}

func NewSQLCAdapter(db *sqlx.DB) *SQLCAdapter {
    return &SQLCAdapter{db: db}
}

// 实现完整的 DBTX 接口
func (a *SQLCAdapter) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
    result, err := a.db.ExecContext(ctx, query, args...)
    if err != nil {
        return pgconn.CommandTag{}, err
    }
    
    rowsAffected, _ := result.RowsAffected()
    return pgconn.NewCommandTag(fmt.Sprintf("ROWS %d", rowsAffected)), nil
}

func (a *SQLCAdapter) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
    rows, err := a.db.QueryxContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    
    return &SQLCRowAdapter{rows: rows, isRow: false}, nil
}

func (a *SQLCAdapter) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
    return &SQLCRowAdapter{row: a.db.QueryRowxContext(ctx, query, args...), isRow: true}
}

// 实现完整的 Rows/Row 适配器
type SQLCRowAdapter struct {
    rows  *sqlx.Rows
    row   *sqlx.Row
    isRow bool
}

// 实现所有必需的方法...
```

## 🛠️ 实施步骤

### 步骤 1: 完善适配器
```bash
# 1. 创建完整的 SQLC 适配器
# 2. 测试适配器功能
# 3. 验证类型转换
```

### 步骤 2: 迁移用户模块
```bash
# 1. 创建 SQLC 用户仓库
# 2. 更新模块初始化
# 3. 运行测试验证
# 4. 性能基准测试
```

### 步骤 3: 迁移其他模块
```bash
# 1. 优惠券模块迁移
# 2. 支付模块迁移
# 3. 时刻模块迁移
# 4. 集成测试
```

### 步骤 4: 清理和优化
```bash
# 1. 删除旧的 GORM/SQLX 代码
# 2. 更新文档
# 3. 性能优化
# 4. 监控和日志
```

## 📈 预期收益

### 性能提升
- **查询性能**: 10-20% 提升
- **内存使用**: 减少反射开销
- **编译时间**: 类型检查优化

### 开发体验
- **类型安全**: 编译时错误检查
- **IDE 支持**: 完整的代码补全
- **维护性**: SQL 集中管理

### 代码质量
- **错误减少**: SQL 语法错误提前发现
- **测试性**: 生成的代码易于测试
- **一致性**: 统一的数据库访问模式

## 🎯 成功指标

### 技术指标
- [ ] 所有模块使用 SQLC
- [ ] 编译时类型检查通过
- [ ] 性能基准测试达标
- [ ] 单元测试覆盖率 > 90%

### 业务指标
- [ ] API 响应时间改善
- [ ] 错误率降低
- [ ] 开发效率提升
- [ ] 代码维护成本降低

## 🚨 风险评估

### 技术风险
- **适配器复杂性**: 需要仔细实现 DBTX 接口
- **类型转换**: pgtype 到 domain model 的转换
- **性能回归**: 不正确的适配器实现

### 缓解措施
- **渐进式迁移**: 逐模块迁移，降低风险
- **充分测试**: 每个阶段都有完整的测试
- **性能监控**: 实时监控性能指标
- **回滚计划**: 保留原有代码作为备份

## 📅 时间计划

### 第 1 周: 适配器完善
- [ ] 完成 SQLC 适配器实现
- [ ] 适配器单元测试
- [ ] 性能基准测试

### 第 2 周: 用户模块迁移
- [ ] SQLC 用户仓库实现
- [ ] 类型转换函数
- [ ] 集成测试

### 第 3 周: 其他模块迁移
- [ ] 优惠券模块迁移
- [ ] 支付模块迁移
- [ ] 时刻模块迁移

### 第 4 周: 测试和优化
- [ ] 完整的集成测试
- [ ] 性能优化
- [ ] 文档更新

## 🎉 总结

这个迁移计划将帮助项目完全迁移到 SQLC，享受类型安全、性能提升和更好的开发体验。通过渐进式迁移和充分的测试，我们可以确保迁移过程平稳进行，同时最大化 SQLC 的收益。

**关键成功因素**:
1. 完整的适配器实现
2. 仔细的类型转换
3. 充分的测试覆盖
4. 渐进式迁移策略

**预期结果**: 100% SQLC 覆盖，显著提升的类型安全性和性能 🚀
