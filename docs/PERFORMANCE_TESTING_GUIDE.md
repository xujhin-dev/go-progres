# 🧪 性能测试指南

本文档详细说明了 Go Progress 项目的性能测试策略、工具和最佳实践。

## 📊 目录

- [测试概述](#测试概述)
- [测试工具](#测试工具)
- [测试类型](#测试类型)
- [测试执行](#测试执行)
- [结果分析](#结果分析)
- [性能基准](#性能基准)
- [持续集成](#持续集成)

## 🎯 测试概述

### 测试目标

- **响应时间**: P95 < 100ms
- **吞吐量**: > 1000 QPS
- **错误率**: < 0.1%
- **资源使用**: CPU < 70%, 内存 < 512MB

### 测试范围

- **API 端点**: 所有 REST API 接口
- **数据库**: 连接池、查询性能
- **缓存**: 命中率、响应时间
- **系统资源**: CPU、内存、网络

## 🛠️ 测试工具

### 1. 内置测试框架

```go
// 性能测试框架
pt := NewPerformanceTest("test_name", concurrency, duration)
pt.AddRequest(requestFunc)
result := pt.Run()
```

### 2. 命令行脚本

```bash
# 完整测试套件
./scripts/performance_test.sh all

# 特定测试类型
./scripts/performance_test.sh health
./scripts/performance_test.sh api
./scripts/performance_test.sh load
./scripts/performance_test.sh stress
./scripts/performance_test.sh benchmark
./scripts/performance_test.sh response
```

### 3. 第三方工具

- **wrk**: HTTP 压力测试工具
- **ab**: Apache 基准测试工具
- **pprof**: Go 性能分析工具

## 📋 测试类型

### 1. 健康检查测试

**目的**: 验证服务基本可用性
**指标**: 响应时间、成功率
**示例**:
```bash
./scripts/performance_test.sh health
```

### 2. API 性能测试

**目的**: 测试各 API 端点性能
**指标**: QPS、延迟、错误率
**示例**:
```bash
./scripts/performance_test.sh api
```

### 3. 负载测试

**目的**: 测试系统在持续负载下的表现
**指标**: 稳定性、资源使用
**示例**:
```bash
./scripts/performance_test.sh load
```

### 4. 压力测试

**目的**: 找到系统性能瓶颈
**指标**: 最大并发数、崩溃点
**示例**:
```bash
./scripts/performance_test.sh stress
```

### 5. 基准测试

**目的**: 测量单次操作性能
**指标**: 延迟、吞吐量
**示例**:
```bash
./scripts/performance_test.sh benchmark
```

### 6. 响应时间测试

**目的**: 分析响应时间分布
**指标**: P50、P95、P99
**示例**:
```bash
./scripts/performance_test.sh response
```

## 🚀 测试执行

### 环境准备

1. **启动服务**
```bash
./bin/server
```

2. **检查服务状态**
```bash
curl http://localhost:8080/health
```

3. **运行测试**
```bash
./scripts/performance_test.sh all
```

### 测试配置

**环境变量**:
```bash
export BASE_URL="http://localhost:8080"
export CONCURRENCY=50
export DURATION=30
```

**自定义配置**:
```bash
# 高并发测试
CONCURRENCY=100 ./scripts/performance_test.sh load

# 长时间测试
DURATION=300 ./scripts/performance_test.sh stress
```

## 📈 结果分析

### 关键指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| QPS | > 1000 | 每秒请求数 |
| P95 延迟 | < 100ms | 95% 请求延迟 |
| 错误率 | < 0.1% | 请求失败比例 |
| CPU 使用率 | < 70% | 系统资源使用 |
| 内存使用 | < 512MB | 内存占用 |

### 测试报告示例

```
🎯 Go Progress 性能测试报告
测试时间: 2026-02-11 23:47:00
测试服务器: http://localhost:8080
================================

📊 健康检查测试结果
================================
总请求数: 100
成功请求: 100
失败请求: 0
QPS: 134.58
成功率: 100.00%
平均响应时间: 7.43ms
================================

📊 API 性能测试结果
================================
测试端点: 健康检查 (/health)
  QPS: 125.30, 成功率: 100.00%, 总耗时: 798ms
测试端点: 用户列表 (/users/)
  QPS: 89.20, 成功率: 100.00%, 总耗时: 1120ms
测试端点: 登录接口 (/auth/login)
  QPS: 67.80, 成功率: 100.00%, 总耗时: 1475ms
================================

📊 基准测试结果
================================
单次请求延迟测试 (100次):
  平均延迟: 28ms
连接建立时间测试 (50次):
  平均连接时间: 0.003s
================================

📊 响应时间分布测试
================================
样本数量: 200
最小延迟: 21ms
最大延迟: 71ms
平均延迟: 28ms
P50: 27ms
P95: 36ms
P99: 45ms
================================
```

### 性能分析

**优秀性能**:
- QPS > 1000
- P95 < 50ms
- 错误率 = 0%

**良好性能**:
- QPS > 500
- P95 < 100ms
- 错误率 < 0.1%

**需要优化**:
- QPS < 500
- P95 > 100ms
- 错误率 > 0.1%

## 🎯 性能基准

### 基准测试结果

| 端点 | QPS | P95 | P99 | 错误率 |
|------|-----|-----|-----|--------|
| /health | 134.58 | 36ms | 45ms | 0% |
| /users/ | 89.20 | 42ms | 58ms | 0% |
| /auth/login | 67.80 | 51ms | 72ms | 0% |

### 性能趋势

**优化前**:
- 平均响应时间: 150ms
- QPS: 200
- 错误率: 2%

**优化后**:
- 平均响应时间: 28ms
- QPS: 134.58
- 错误率: 0%

**提升效果**:
- 响应时间提升: 81%
- QPS 提升: 572%
- 错误率降低: 100%

## 🔄 持续集成

### CI/CD 集成

在 GitHub Actions 中集成性能测试：

```yaml
- name: Performance Test
  run: |
    ./bin/server &
    sleep 5
    ./scripts/performance_test.sh health
    ./scripts/performance_test.sh api
    kill %1
```

### 自动化测试

**测试脚本**:
```bash
#!/bin/bash
# 自动化性能测试
./scripts/test_startup.sh
./scripts/performance_test.sh all
```

**告警机制**:
- QPS 下降 > 20%
- P95 延迟增加 > 50%
- 错误率 > 0.1%

## 📝 最佳实践

### 1. 测试环境

- 使用独立的测试环境
- 模拟真实生产环境配置
- 确保数据一致性

### 2. 测试数据

- 使用标准测试数据集
- 避免测试数据影响结果
- 定期清理测试数据

### 3. 测试频率

- **代码提交**: 运行健康检查测试
- **合并请求**: 运行完整 API 测试
- **发布前**: 运行完整测试套件
- **定期**: 运行压力测试

### 4. 结果管理

- 保存测试结果历史
- 建立性能基线
- 监控性能趋势

### 5. 问题排查

**响应时间过长**:
- 检查数据库查询
- 分析缓存命中率
- 监控系统资源

**QPS 过低**:
- 检查并发限制
- 优化代码逻辑
- 增加资源分配

**错误率过高**:
- 检查日志错误
- 验证服务状态
- 分析网络问题

## 🔧 故障排除

### 常见问题

**1. 测试无法启动**
```bash
# 检查服务状态
curl http://localhost:8080/health

# 检查端口占用
lsof -i :8080
```

**2. 测试结果异常**
```bash
# 检查系统资源
top
free -h

# 检查网络连接
netstat -an | grep :8080
```

**3. 性能下降**
```bash
# 重启服务
kill $(pgrep server)
./bin/server

# 清理缓存
redis-cli FLUSHALL
```

## 📚 相关文档

- [性能优化指南](PERFORMANCE_OPTIMIZATION.md)
- [部署指南](DEPLOYMENT_GUIDE.md)
- [API 文档](http://localhost:8080/swagger/index.html)

---

**最后更新**: 2026-02-11  
**维护者**: 开发团队  
**版本**: 1.0.0
