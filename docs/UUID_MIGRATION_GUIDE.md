# UUID 迁移指南

## 概述

本次重构将所有主键从 `int/uint` 类型迁移到 `UUID string` 类型，以提升系统的可扩展性和安全性。

## 变更内容

### 1. 模型层变更

#### BaseModel (pkg/model/base.go)

- ID 类型：`uint` → `string (UUID)`
- 添加了 `BeforeCreate` 钩子自动生成 UUID

#### User Model (internal/domain/user/model/user.go)

- 已经使用 UUID，无需修改

#### 其他模型

- **Coupon/UserCoupon**: ID 和外键 UserID/CouponID 改为 `string`
- **Order**: ID 和 UserID 改为 `string`
- **Post**: ID 和 UserID 改为 `string`
- **Comment**: ID、PostID、UserID、ParentID、RootID 改为 `string`
- **Like**: ID、UserID、TargetID 改为 `string`
- **Topic**: ID 改为 `string`

### 2. Service 层变更

所有 Service 接口中的 `userID uint` 参数改为 `userID string`：

- `UserService`: GetUser, UpdateUser, DeleteUser, UpgradeMember
- `CouponService`: ClaimCoupon, SendCouponToUser
- `PaymentService`: CreateOrder
- `MomentService`: PublishPost, AuditPost, AddComment, GetPostComments, ToggleLike, DeleteTopic

### 3. Repository 层变更

所有 Repository 接口中的 ID 参数从 `uint` 改为 `string`：

- `UserRepository`: GetByID, UpdateMemberStatus
- `CouponRepository`: GetByID, DecreaseStock, HasUserClaimed
- `MomentRepository`: GetPostByID, UpdatePostStatus, GetCommentByID, GetCommentsByPostID, DeleteLike, HasLiked, DeleteTopic

### 4. Handler 层变更

- 移除了 `strconv.ParseUint` 的 ID 转换逻辑
- `getUserIdFromContext` 函数返回类型改为 `string`
- URL 参数直接作为 string 使用，无需转换

### 5. JWT 工具变更

- `Claims.UserID` 已经是 `string` 类型
- `GenerateToken` 和 `ParseToken` 已支持 string UserID

### 6. Worker 层变更

- `CouponTask.UserID` 和 `CouponTask.CouponID` 改为 `string`

## 数据库迁移

### 迁移脚本

新增迁移文件：

- `migrations/000006_refactor_to_uuid.up.sql`
- `migrations/000006_refactor_to_uuid.down.sql`

### 重要提示

⚠️ **本次迁移将清空所有数据！**

迁移脚本会：

1. 删除所有现有表
2. 重新创建使用 UUID 的表结构
3. 启用 PostgreSQL UUID 扩展

### 执行迁移

```bash
# 运行迁移
go run cmd/migrate/main.go

# 或使用 migrate 工具
migrate -path migrations -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" up
```

### 回滚

如需回滚到旧版本：

```bash
migrate -path migrations -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" down 1
```

## 测试建议

### 1. 单元测试

更新所有涉及 ID 的测试用例：

- 使用 UUID 字符串而非整数
- 测试 UUID 生成逻辑

### 2. 集成测试

- 测试用户注册/登录流程
- 测试优惠券领取（包括 Redis 缓存）
- 测试订单创建和支付回调
- 测试动态发布和评论功能

### 3. API 测试

更新 API 测试脚本中的 ID 格式：

```bash
# 旧格式
curl -X GET http://localhost:8080/api/users/123

# 新格式
curl -X GET http://localhost:8080/api/users/550e8400-e29b-41d4-a716-446655440000
```

## 性能考虑

### UUID vs Int 性能对比

**优势：**

- 分布式系统友好，无需中心化 ID 生成
- 更好的安全性（ID 不可预测）
- 支持离线生成

**劣势：**

- 存储空间增加（16 bytes vs 4/8 bytes）
- 索引性能略有下降（但使用 UUID v4 影响较小）

### 优化建议

1. 使用 PostgreSQL 的 `gen_random_uuid()` 函数
2. 为高频查询字段添加索引
3. 考虑使用 UUID v7（时间有序）以提升索引性能

## 兼容性

### 前端适配

前端需要更新：

1. ID 字段类型从 `number` 改为 `string`
2. URL 路径参数使用 UUID 字符串
3. 表单验证规则（如果有 ID 输入）

### 第三方集成

检查以下集成点：

- Redis 缓存键（已更新为使用 string）
- 推送通知（已更新 AccountID 为 string）
- 支付回调（Order 查询已更新）

## 常见问题

### Q: 为什么选择 UUID 而不是 Snowflake ID？

A: UUID 更简单，无需维护额外的 ID 生成服务，适合中小规模应用。

### Q: 如何处理现有数据？

A: 本次迁移采用"重置数据库"策略，不保留旧数据。如需保留，请手动导出并转换。

### Q: UUID 会影响性能吗？

A: 对于大多数应用，影响可以忽略。PostgreSQL 对 UUID 有良好支持。

### Q: 可以使用有序 UUID 吗？

A: 可以，考虑使用 UUID v7 或 ULID 以获得更好的索引性能。

## 后续优化

1. 考虑使用 UUID v7 替代 v4
2. 评估是否需要为特定表保留自增 ID
3. 监控数据库性能指标
4. 优化高频查询的索引策略

## 完成检查清单

- [x] 模型层 ID 类型修改
- [x] Service 接口签名更新
- [x] Repository 接口签名更新
- [x] Handler 层 ID 处理逻辑更新
- [x] JWT Claims 类型确认
- [x] Worker 任务结构更新
- [x] 数据库迁移脚本创建
- [ ] 单元测试更新
- [ ] 集成测试更新
- [ ] API 文档更新
- [ ] 前端代码适配
- [ ] 性能测试
- [ ] 生产环境部署计划

## 联系方式

如有问题，请联系开发团队。
