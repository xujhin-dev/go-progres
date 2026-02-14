# 🚀 性能优化指南

本文档详细说明了 Go Progress 项目的性能优化策略和实施细节。

## 📊 目录

- [优化概览](#优化概览)
- [数据库优化](#数据库优化)
- [缓存策略](#缓存策略)
- [并发控制](#并发控制)
- [性能监控](#性能监控)
- [HTTP 优化](#http-优化)
- [性能测试](#性能测试)
- [最佳实践](#最佳实践)

## 🎯 优化概览

### 优化目标
- **响应时间**: P95 < 100ms
- **吞吐量**: > 1000 QPS
- **内存使用**: < 512MB
- **CPU 使用率**: < 70%
- **错误率**: < 0.1%

### 优化策略
1. **数据库层**: 连接池优化、查询优化、索引优化
2. **缓存层**: 多级缓存、缓存预热、缓存失效策略
3. **应用层**: 并发控制、资源池化、异步处理
4. **网络层**: HTTP 优化、压缩、限流
5. **监控层**: 实时指标、性能追踪、告警

## 🗄️ 数据库优化

### 连接池优化

```go
// PostgreSQL 连接池配置
sqlDB.SetMaxOpenConns(100)        // 最大连接数
sqlDB.SetMaxIdleConns(10)         // 最大空闲连接数
sqlDB.SetConnMaxLifetime(time.Hour)    // 连接最大生命周期
sqlDB.SetConnMaxIdleTime(time.Minute * 30) // 空闲连接超时
```

**优化效果**:
- 减少连接建立开销
- 提高连接复用率
- 避免连接泄漏

### 查询优化

```go
// GORM 配置优化
gormConfig := &gorm.Config{
    PrepareStmt: true,  // 预编译 SQL 缓存
    Logger: logger.Default.LogMode(logger.Info), // 生产环境改为 Warn
}
```

**优化建议**:
- 使用预编译语句
- 避免 N+1 查询
- 合理使用索引
- 分页查询优化

### Redis 连接优化

```go
// Redis 连接池配置
rdb := redis.NewClient(&redis.Options{
    PoolSize:     50,              // 连接池大小
    MinIdleConns: 10,              // 最小空闲连接数
    MaxRetries:   3,               // 最大重试次数
    DialTimeout:  time.Second * 5,  // 连接超时
    ReadTimeout:  time.Second * 3,  // 读取超时
    WriteTimeout: time.Second * 3,  // 写入超时
})
```

## 💾 缓存策略

### 多级缓存架构

```
请求 → 内存缓存 → Redis 缓存 → 数据库
```

### 缓存实现

```go
// 缓存服务接口
type CacheService interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
    Delete(ctx context.Context, key string) error
    InvalidatePattern(ctx context.Context, pattern string) error
}
```

### 缓存策略

**缓存键设计**:
- 用户缓存: `user:{id}`
- 用户列表: `user_list:{page}:{limit}`
- 热点数据: `hot:{type}:{id}`

**TTL 策略**:
- 用户信息: 2小时
- 用户列表: 30分钟
- 热点数据: 1小时

**缓存预热**:
```go
// 预热热门用户数据
func (s *CachedUserService) WarmupCache(ctx context.Context) error {
    popularUsers := []string{"1", "2", "3"}
    for _, userID := range popularUsers {
        user, err := s.repo.GetByID(userID)
        if err != nil {
            continue
        }
        s.cache.Set(ctx, s.getUserCacheKey(userID), user, UserCacheTTL)
    }
    return nil
}
```

## ⚡ 并发控制

### 协程池

```go
// 工作池实现
type GoRoutinePool struct {
    workerCount int
    taskQueue   chan Task
    quit        chan struct{}
    wg          sync.WaitGroup
}

// 创建协程池
pool := NewGoRoutinePool(50, 1000) // 50个工作协程，队列大小1000
```

### 限流器

```go
// 令牌桶限流器
type RateLimiter struct {
    tokens     int
    maxTokens  int
    refillRate int
    lastRefill time.Time
}

// 使用示例
rateLimiter := NewRateLimiter(1000, 2000) // 1000 req/s, burst 2000
if !rateLimiter.Allow() {
    return errors.New("rate limit exceeded")
}
```

### 熔断器

```go
// 熔断器实现
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    failures     int
    state        CircuitState
}

// 使用示例
circuitBreaker := NewCircuitBreaker(10, time.Minute*5)
err := circuitBreaker.Call(func() error {
    // 执行业务逻辑
    return doSomething()
})
```

## 📈 性能监控

### Prometheus 指标

```go
// HTTP 指标
http_requests_total{method, endpoint, status}
http_request_duration_seconds{method, endpoint}
http_request_size_bytes{method, endpoint}
http_response_size_bytes{method, endpoint}

// 数据库指标
db_connections_active
db_connections_idle
db_query_duration_seconds{operation, table}
db_queries_total{operation, table, status}
db_errors_total{operation, error_type}

// 缓存指标
cache_hits_total{cache_type, key_prefix}
cache_misses_total{cache_type, key_prefix}
cache_operation_duration_seconds{operation, cache_type}

// 系统指标
active_goroutines
memory_usage_bytes
```

### 性能追踪

```go
// 性能跟踪器
type PerformanceTracker struct {
    startTime time.Time
    operation string
    collector *MetricsCollector
}

// 使用示例
tracker := NewPerformanceTracker(collector, "db_query", "select", "users")
defer tracker.Finish()
```

### Grafana 仪表板

- **请求概览**: QPS、响应时间、错误率
- **数据库监控**: 连接数、查询时间、错误统计
- **缓存监控**: 命中率、操作延迟、存储使用
- **系统监控**: CPU、内存、goroutine 数量

## 🌐 HTTP 优化

### 中间件优化

```go
// 性能中间件
func PerformanceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // 限流检查
        if !rateLimiter.Allow() {
            c.JSON(429, gin.H{"error": "rate limit exceeded"})
            c.Abort()
            return
        }
        
        // 熔断器检查
        if circuitBreaker.State() == StateOpen {
            c.JSON(503, gin.H{"error": "service unavailable"})
            c.Abort()
            return
        }
        
        c.Next()
        
        // 记录指标
        duration := time.Since(start)
        recordMetrics(c, duration)
    }
}
```

### Gzip 压缩

```go
// Gzip 中间件
func GzipMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if shouldCompress(c.Request) {
            gz := gzip.NewWriter(c.Writer)
            defer gz.Close()
            c.Writer = &gzipWriter{c.Writer, gz}
        }
        c.Next()
    }
}
```

### 连接复用

```nginx
# Nginx 配置
upstream app_backend {
    least_conn;
    server app:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

server {
    location / {
        proxy_pass http://app_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
    }
}
```

## 🧪 性能测试

### 性能测试框架

项目提供了完整的性能测试框架，支持多种测试类型：

```go
// 性能测试
pt := NewPerformanceTest("test_name", 50, time.Second*30)
pt.AddRequest(requestFunc)
result := pt.Run()
```

### 测试工具

**1. 命令行工具**
```bash
# 运行完整测试套件
./scripts/performance_test.sh all

# 运行特定测试
./scripts/performance_test.sh health
./scripts/performance_test.sh api
./scripts/performance_test.sh load
./scripts/performance_test.sh benchmark
```

**2. 测试类型**
- **健康检查测试**: 基础连通性测试
- **API 性能测试**: 各端点性能测试
- **负载测试**: 持续负载测试
- **压力测试**: 渐进式压力测试
- **基准测试**: 单次请求延迟测试
- **响应时间测试**: 延迟分布测试

### 测试结果分析

**性能指标**:
- QPS (每秒请求数)
- 响应时间 (平均、P50、P95、P99)
- 成功率/错误率
- 吞吐量

**测试结果示例**:
```
📊 健康检查测试结果
================================
总请求数: 100
成功请求: 100
失败请求: 0
QPS: 134.58
成功率: 100.00%
平均响应时间: 7.43ms
================================
```

### 压力测试工具

```bash
# 使用 wrk 进行专业压力测试
wrk -t12 -c400 -d30s http://localhost:8080/health

# 使用 ab 进行测试
ab -n 10000 -c 100 http://localhost:8080/api/users
```

### 基准测试

```go
func BenchmarkGetUser(b *testing.B) {
    service := setupUserService()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := service.GetUser("1")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 性能分析

```bash
# CPU 性能分析
go tool pprof http://localhost:8080/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap

# 协程分析
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

## 📋 最佳实践

### 1. 数据库优化

- ✅ 使用连接池
- ✅ 预编译 SQL 语句
- ✅ 合理设计索引
- ✅ 避免 N+1 查询
- ✅ 使用分页查询

### 2. 缓存优化

- ✅ 多级缓存策略
- ✅ 合理设置 TTL
- ✅ 缓存预热
- ✅ 缓存失效策略
- ✅ 监控缓存命中率

### 3. 并发优化

- ✅ 使用协程池
- ✅ 实施限流策略
- ✅ 使用熔断器
- ✅ 避免协程泄漏
- ✅ 合理设置超时

### 4. 监控优化

- ✅ 关键指标监控
- ✅ 实时告警
- ✅ 性能追踪
- ✅ 日志分析
- ✅ 定期性能评估

## 🔧 性能调优清单

### 日常检查

- [ ] 检查响应时间 P95 < 100ms
- [ ] 检查 QPS > 1000
- [ ] 检查错误率 < 0.1%
- [ ] 检查缓存命中率 > 80%
- [ ] 检查数据库连接池使用率

### 定期优化

- [ ] 分析慢查询日志
- [ ] 优化数据库索引
- [ ] 调整缓存策略
- [ ] 更新监控指标
- [ ] 进行压力测试

### 容量规划

- [ ] 评估当前负载
- [ ] 预测增长趋势
- [ ] 规划资源扩容
- [ ] 制定应急预案
- [ ] 更新架构设计

## 📊 性能指标参考

| 指标 | 目标值 | 当前值 | 状态 |
|------|--------|--------|------|
| P95 响应时间 | < 100ms | TBD | 🟡 |
| QPS | > 1000 | TBD | 🟡 |
| 错误率 | < 0.1% | TBD | 🟡 |
| 缓存命中率 | > 80% | TBD | 🟡 |
| 内存使用 | < 512MB | TBD | 🟡 |
| CPU 使用率 | < 70% | TBD | 🟡 |

## 🚀 下一步计划

1. **性能基准测试**: 建立性能基准线
2. **监控完善**: 添加更多业务指标
3. **自动化测试**: 集成性能测试到 CI/CD
4. **容量规划**: 制定扩容策略
5. **持续优化**: 定期性能评估和优化

---

**最后更新**: 2026-02-11  
**维护者**: 开发团队  
**版本**: 1.0.0
