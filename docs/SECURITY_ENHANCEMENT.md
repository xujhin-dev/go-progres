# 🔒 安全性增强指南

本文档详细说明了 Go Progress 项目的安全性增强策略和实施细节。

## 📊 目录

- [安全概览](#安全概览)
- [API 限流增强](#api-限流增强)
- [输入验证和过滤](#输入验证和过滤)
- [JWT 安全机制](#jwt-安全机制)
- [安全头和防护](#安全头和防护)
- [安全监控和日志](#安全监控和日志)
- [权限控制系统](#权限控制系统)
- [安全最佳实践](#安全最佳实践)

## 🎯 安全概览

### 安全目标

- **访问控制**: 基于角色和权限的细粒度访问控制
- **数据保护**: 输入验证、XSS/SQL 注入防护
- **认证安全**: JWT 令牌管理、黑名单机制
- **传输安全**: HTTPS、安全头设置
- **监控审计**: 安全事件记录、实时监控
- **防护机制**: 限流、熔断、IP 白名单

### 安全架构

```
┌─────────────────────────────────────────────────────────┐
│                    安全防护层                              │
├─────────────────────────────────────────────────────────┤
│  安全头  │  CORS  │  限流  │  输入验证  │  审计日志      │
├─────────────────────────────────────────────────────────┤
│                   认证授权层                                │
├─────────────────────────────────────────────────────────┤
│  JWT 认证  │  权限控制  │  角色管理  │  策略引擎        │
├─────────────────────────────────────────────────────────┤
│                   应用安全层                                │
├─────────────────────────────────────────────────────────┤
│  XSS 防护  │  SQL 注入防护  │  数据验证  │  错误处理      │
├─────────────────────────────────────────────────────────┤
│                   监控告警层                                │
├─────────────────────────────────────────────────────────┤
│  安全事件  │  异常检测  │  告警通知  │  审计报告        │
└─────────────────────────────────────────────────────────┘
```

## 🚦 API 限流增强

### 限流策略

**令牌桶算法**:
```go
// 令牌桶限流器
type TokenBucket struct {
    cache  cache.CacheService
    limits map[string]Limit
}

// 使用示例
rateLimiter := NewTokenBucket(cache)
rateLimiter.SetLimit("api", Limit{
    Rate:   100,  // 100 requests/second
    Burst:  200,  // 200 burst
    Window: time.Second,
})
```

**滑动窗口日志**:
```go
// 滑动窗口限流器
slidingWindow := NewSlidingWindowLog(cache, Limit{
    Rate:   50,
    Burst:  100,
    Window: time.Second,
})
```

**多级限流**:
```go
// 全局限流 + 用户限流 + IP 限流
multiLimiter := NewMultiRateLimiter()
multiLimiter.AddLimiter("global", globalLimiter)
multiLimiter.AddLimiter("user", userLimiter)
multiLimiter.AddLimiter("ip", ipLimiter)
```

### 限流配置

```go
// 默认限流配置
config := DefaultRateLimitConfig()
config.Global = Limit{Rate: 1000, Burst: 2000, Window: time.Second}
config.User = Limit{Rate: 100, Burst: 200, Window: time.Second}
config.IP = Limit{Rate: 50, Burst: 100, Window: time.Second}
config.Endpoint = map[string]Limit{
    "auth": {Rate: 10, Burst: 20, Window: time.Second},
    "upload": {Rate: 5, Burst: 10, Window: time.Second},
}
```

## 🔍 输入验证和过滤

### 验证器类型

**字符串验证器**:
```go
validator := NewStringValidator(1, 100, true)
validator.SetPattern(`^[a-zA-Z0-9]+$`)

err := validator.Validate("username")
if err != nil {
    // 处理验证错误
}
```

**邮箱验证器**:
```go
emailValidator := NewEmailValidator(true)
err := emailValidator.Validate("user@example.com")
```

**手机号验证器**:
```go
phoneValidator := NewPhoneValidator(true, "CN")
err := phoneValidator.Validate("13800138000")
```

### 输入过滤

**XSS 防护**:
```go
xssProtection := NewXSSProtection()
cleanHTML := xssProtection.SanitizeHTML(userInput)
```

**SQL 注入防护**:
```go
sqlProtection := NewSQLInjectionProtection()
if sqlProtection.CheckSQLInjection(input) {
    // 检测到潜在的 SQL 注入
}
cleanInput := sqlProtection.SanitizeSQL(input)
```

**综合输入过滤**:
```go
inputFilter := NewInputFilter(1000, false)
filteredInput, err := inputFilter.FilterInput(userInput)
```

## 🔐 JWT 安全机制

### JWT 安全特性

**令牌对生成**:
```go
jwtSecurity := NewJWTSecurity(secret, issuer, cache)

// 生成访问令牌和刷新令牌
accessToken, refreshToken, err := jwtSecurity.GenerateTokenPair(
    userID, 
    role, 
    permissions,
)
```

**令牌验证**:
```go
claims, err := jwtSecurity.ValidateToken(tokenString)
if err != nil {
    // 令牌无效
}
```

**令牌刷新**:
```go
newAccessToken, newRefreshToken, err := jwtSecurity.RefreshToken(refreshToken)
```

**令牌撤销**:
```go
// 撤销单个令牌
err := jwtSecurity.RevokeToken(tokenString)

// 撤销用户所有令牌
err := jwtSecurity.RevokeUserTokens(userID)
```

### 令牌黑名单

```go
// 令牌自动加入黑名单
jwtSecurity.addToBlacklist(tokenID, expiresAt)

// 检查令牌是否在黑名单中
if jwtSecurity.isTokenBlacklisted(tokenID) {
    return fmt.Errorf("token is blacklisted")
}
```

## 🛡️ 安全头和防护

### 安全头设置

```go
// 安全中间件配置
securityConfig := SecurityConfig{
    EnableCSRF:      true,
    EnableXSS:       true,
    EnableCORS:      true,
    EnableRateLimit: true,
    XSSProtection:   "1; mode=block",
    ContentType:     "nosniff",
    FrameOptions:    "DENY",
    HSTS:           true,
    HSTSMaxAge:      31536000,
}

securityMiddleware := NewSecurityMiddleware(config, jwtSecurity, rateLimiter, inputFilter)
```

### 安全头列表

| 安全头 | 作用 | 默认值 |
|--------|------|--------|
| X-XSS-Protection | XSS 保护 | "1; mode=block" |
| X-Content-Type-Options | 内容类型嗅探保护 | "nosniff" |
| X-Frame-Options | 点击劫持保护 | "DENY" |
| Content-Security-Policy | 内容安全策略 | "default-src 'self'" |
| Strict-Transport-Security | HTTPS 强制 | "max-age=31536000" |
| Referrer-Policy | 引用策略 | "strict-origin-when-cross-origin" |

### CORS 配置

```go
corsOrigins := []string{
    "http://localhost:3000",
    "http://localhost:8080",
    "https://yourdomain.com",
}

corsMethods := []string{
    "GET", "POST", "PUT", "DELETE", "OPTIONS",
}

corsHeaders := []string{
    "Origin", "Content-Type", "Accept", 
    "Authorization", "X-Request-ID",
}
```

## 📊 安全监控和日志

### 安全事件类型

```go
const (
    EventLogin          SecurityEventType = "login"
    EventLogout         SecurityEventType = "logout"
    EventTokenExpired   SecurityEventType = "token_expired"
    EventTokenRevoked  SecurityEventType = "token_revoked"
    EventRateLimit      SecurityEventType = "rate_limit"
    EventSuspicious    SecurityEventType = "suspicious"
    EventXSS            SecurityEventType = "xss"
    EventSQLInjection  SecurityEventType = "sql_injection"
    EventCSRF           SecurityEventType = "csrf"
    EventUnauthorized  SecurityEventType = "unauthorized"
    EventForbidden     SecurityEventType = "forbidden"
)
```

### 安全监控

```go
// 创建安全监控器
securityMonitor := NewSecurityMonitor(cache, metricsCollector, logger)

// 记录安全事件
securityMonitor.RecordEvent(SecurityEvent{
    Type:      EventUnauthorized,
    Level:     LevelWarning,
    Source:    "api",
    UserID:    userID,
    IP:        clientIP,
    Path:      requestPath,
    Method:    requestMethod,
    Status:    statusCode,
    Message:   "Unauthorized access attempt",
})
```

### 告警机制

```go
// 邮件告警处理器
emailHandler := NewEmailAlertHandler(
    "smtp.example.com", 587,
    "user", "password", 
    "security@example.com",
    []string{"admin@example.com"},
)

// Slack 告警处理器
slackHandler := NewSlackAlertHandler(
    "https://hooks.slack.com/...",
    "#security-alerts",
)

securityMonitor.AddAlertHandler(emailHandler)
securityMonitor.AddAlertHandler(slackHandler)
```

### 安全指标

```go
// 获取安全指标
metrics := securityMonitor.GetMetrics()

// 生成安全报告
report := securityMonitor.GenerateReport(time.Hour * 24)
```

## 🔑 权限控制系统

### RBAC 模型

**角色定义**:
```go
const (
    RoleUser      Role = "user"
    RoleModerator Role = "moderator"
    RoleAdmin     Role = "admin"
    RoleSuperAdmin Role = "super_admin"
)
```

**权限定义**:
```go
const (
    PermissionUserRead    Permission = "user:read"
    PermissionUserWrite   Permission = "user:write"
    PermissionUserDelete  Permission = "user:delete"
    PermissionAdminSystem Permission = "admin:system"
)
```

### 权限检查

```go
// 创建 RBAC 实例
rbac := NewRBAC(cache)

// 分配角色
rbac.AssignRole(userID, RoleAdmin)

// 检查权限
hasPermission, err := rbac.HasPermission(userID, PermissionUserRead)

// 检查角色
hasRole, err := rbac.HasRole(userID, RoleAdmin)
```

### 权限中间件

```go
// 权限检查中间件
permissionMiddleware := NewPermissionMiddleware(rbac, PermissionUserRead)
router.Use("/users", permissionMiddleware.Middleware())

// 角色检查中间件
roleMiddleware := NewRoleMiddleware(rbac, RoleAdmin)
router.Use("/admin", roleMiddleware.Middleware())

// 多权限检查中间件
multiPermissionMiddleware := NewMultiPermissionMiddleware(
    rbac, 
    []Permission{PermissionUserRead, PermissionUserWrite},
    false, // 需要任意权限
)
```

### 策略引擎

```go
// 创建策略引擎
policyEngine := NewPolicyEngine(rbac)

// 添加时间策略
timePolicy := NewTimeBasedPolicy(
    time.Now().Add(-time.Hour*9),  // 9:00
    time.Now().Add(-time.Hour*17), // 17:00
    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
    []int{9, 10, 11, 14, 15, 16, 17},
)
policyEngine.AddPolicy("business_hours", timePolicy)

// 评估策略
decision, err := policyEngine.Evaluate(ctx, PolicyRequest{
    UserID:   userID,
    Resource: "/admin/users",
    Action:   "read",
    Context: map[string]interface{}{
        "ip": clientIP,
    },
})
```

## 📋 安全最佳实践

### 1. 认证安全

- ✅ 使用强密码策略
- ✅ 实现 JWT 令牌刷新机制
- ✅ 设置合理的令牌过期时间
- ✅ 实现令牌黑名单机制
- ✅ 支持多因素认证（MFA）

### 2. 授权安全

- ✅ 实现最小权限原则
- ✅ 使用基于角色的访问控制（RBAC）
- ✅ 定期审查用户权限
- ✅ 实现资源所有权检查
- ✅ 支持策略引擎进行复杂授权

### 3. 输入验证

- ✅ 验证所有用户输入
- ✅ 使用白名单而非黑名单
- ✅ 防护 XSS 攻击
- ✅ 防护 SQL 注入攻击
- ✅ 限制文件上传类型和大小

### 4. 传输安全

- ✅ 强制使用 HTTPS
- ✅ 设置安全 HTTP 头
- ✅ 实现 CORS 策略
- ✅ 使用安全的 Cookie 设置
- ✅ 实现请求签名验证

### 5. 监控审计

- ✅ 记录所有安全事件
- ✅ 实现实时监控告警
- ✅ 定期生成安全报告
- ✅ 监控异常访问模式
- ✅ 实现日志完整性保护

### 6. 防护机制

- ✅ 实现多级限流保护
- ✅ 使用 IP 白名单/黑名单
- ✅ 实现熔断器机制
- ✅ 防护暴力破解攻击
- ✅ 实现自动封禁机制

## 🔧 安全配置示例

### 完整安全配置

```go
// 初始化安全组件
cache := cache.NewRedisCache(redisClient)
metricsCollector := metrics.GetGlobalCollector()
logger := security.NewLogger()

// 创建安全组件
jwtSecurity := security.NewJWTSecurity(secret, issuer, cache)
rateLimiter := security.NewTokenBucket(cache)
inputFilter := security.NewInputFilter(1000, false)
securityMonitor := security.NewSecurityMonitor(cache, metricsCollector, logger)
rbac := security.NewRBAC(cache)

// 配置安全中间件
securityConfig := security.DefaultSecurityConfig()
securityMiddleware := security.NewSecurityMiddleware(
    securityConfig, jwtSecurity, rateLimiter, inputFilter,
)

// 配置监控中间件
monitoringMiddleware := security.NewSecurityMonitoringMiddleware(securityMonitor)

// 配置权限中间件
permissionMiddleware := security.NewPermissionMiddleware(rbac, security.PermissionUserRead)
roleMiddleware := security.NewRoleMiddleware(rbac, security.RoleAdmin)

// 应用中间件
router.Use(securityMiddleware.Middleware())
router.Use(monitoringMiddleware.Middleware())
router.Use("/users", permissionMiddleware.Middleware())
router.Use("/admin", roleMiddleware.Middleware())
```

### 环境变量配置

```bash
# JWT 配置
JWT_SECRET=your-super-secret-key-min-32-characters
JWT_ISSUER=your-app-name
JWT_ACCESS_TOKEN_TTL=24h
JWT_REFRESH_TOKEN_TTL=168h

# 限流配置
RATE_LIMIT_GLOBAL_RATE=1000
RATE_LIMIT_GLOBAL_BURST=2000
RATE_LIMIT_USER_RATE=100
RATE_LIMIT_USER_BURST=200
RATE_LIMIT_IP_RATE=50
RATE_LIMIT_IP_BURST=100

# 安全配置
ENABLE_CSRF=true
ENABLE_XSS_PROTECTION=true
ENABLE_CORS=true
ENABLE_RATE_LIMIT=true
CORS_ORIGINS=http://localhost:3000,https://yourdomain.com

# 监控配置
SECURITY_ALERT_EMAIL=security@example.com
SECURITY_ALERT_WEBHOOK=https://hooks.slack.com/...
```

## 🚨 安全事件响应

### 事件分级

| 级别 | 事件类型 | 响应时间 | 处理方式 |
|------|----------|----------|----------|
| Critical | 系统入侵、数据泄露 | 立即 | 立即阻断、通知管理员 |
| High | 暴力破解、权限提升 | 5分钟 | 临时封禁、加强监控 |
| Medium | 异常访问、可疑操作 | 30分钟 | 记录日志、发送警告 |
| Low | 一般违规、配置错误 | 2小时 | 记录日志、定期审查 |

### 响应流程

1. **检测**: 自动检测安全事件
2. **分析**: 分析事件严重程度
3. **响应**: 根据级别采取相应措施
4. **通知**: 发送告警通知相关人员
5. **记录**: 详细记录事件和处理过程
6. **复盘**: 定期复盘安全事件

## 📚 相关文档

- [性能优化指南](PERFORMANCE_OPTIMIZATION.md)
- [部署指南](DEPLOYMENT_GUIDE.md)
- [API 文档](http://localhost:8080/swagger/index.html)

---

**最后更新**: 2026-02-11  
**维护者**: 开发团队  
**版本**: 1.0.0
