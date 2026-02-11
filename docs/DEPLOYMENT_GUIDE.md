# 🚀 部署指南

本指南详细说明了如何部署 Go Progress 项目到不同环境。

## 📋 目录

- [环境要求](#环境要求)
- [快速部署](#快速部署)
- [详细部署步骤](#详细部署步骤)
- [CI/CD 自动化](#cicd-自动化)
- [监控和维护](#监控和维护)
- [故障排除](#故障排除)

## 🔧 环境要求

### 基础要求
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **Git**: 2.30+
- **操作系统**: Linux (推荐 Ubuntu 20.04+)

### 硬件要求
- **CPU**: 最少 2 核，推荐 4 核
- **内存**: 最少 4GB，推荐 8GB
- **存储**: 最少 20GB，推荐 50GB SSD

### 网络要求
- **端口**: 80, 443, 8080, 5432, 6379, 3000, 9090
- **防火墙**: 确保上述端口可访问

## ⚡ 快速部署

### 1. 克隆项目
```bash
git clone https://github.com/your-org/go-progres.git
cd go-progres
```

### 2. 设置环境
```bash
# 生产环境
./scripts/setup-env.sh production

# 开发环境
./scripts/setup-env.sh development
```

### 3. 配置环境变量
```bash
# 复制并编辑环境配置
cp .env.example .env
vim .env  # 编辑配置
```

### 4. 一键部署
```bash
./scripts/deploy.sh production latest
```

## 📝 详细部署步骤

### 步骤 1: 环境准备

#### 安装 Docker
```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### 创建项目目录
```bash
sudo mkdir -p /opt/go-progres
sudo chown $USER:$USER /opt/go-progres
cd /opt/go-progres
```

### 步骤 2: 配置环境

#### 生产环境配置
```bash
./scripts/setup-env.sh production
```

#### 编辑配置文件
```bash
vim .env
```

**重要配置项：**
```bash
# 数据库配置 (必须修改)
DATABASE_PASSWORD=your_secure_password

# Redis 配置 (建议修改)
REDIS_PASSWORD=your_redis_password

# JWT 配置 (必须修改)
JWT_SECRET=your_jwt_secret_at_least_32_characters

# GitHub 仓库 (必须修改)
GITHUB_REPOSITORY=your-org/go-progres

# 监控配置 (建议修改)
GRAFANA_PASSWORD=your_grafana_password
```

### 步骤 3: 部署服务

#### 方式一：使用部署脚本 (推荐)
```bash
./scripts/deploy.sh production latest
```

#### 方式二：手动部署
```bash
# 拉取最新代码
git pull origin main

# 构建镜像
docker build -t go-progres:latest .

# 启动服务
docker-compose -f docker-compose.prod.yml up -d

# 运行数据库迁移
docker-compose -f docker-compose.prod.yml run --rm app go run cmd/migrate/main.go
```

### 步骤 4: 验证部署

#### 健康检查
```bash
# 检查应用状态
curl http://localhost:8080/health

# 检查服务状态
docker-compose -f docker-compose.prod.yml ps
```

#### 查看日志
```bash
# 应用日志
docker-compose -f docker-compose.prod.yml logs -f app

# 数据库日志
docker-compose -f docker-compose.prod.yml logs -f postgres

# 所有服务日志
docker-compose -f docker-compose.prod.yml logs -f
```

## 🔄 CI/CD 自动化

### GitHub Actions 配置

项目已配置完整的 CI/CD 流水线，包括：

1. **代码检查**: golangci-lint
2. **安全扫描**: Gosec
3. **单元测试**: 带覆盖率报告
4. **集成测试**: 数据库集成测试
5. **镜像构建**: 多架构 Docker 镜像
6. **自动部署**: 生产环境部署

### 触发条件

- **推送**: `main` 分支触发完整流水线
- **推送**: `develop` 分支触发测试和构建
- **Pull Request**: 触发测试和代码检查

### 环境变量配置

在 GitHub 仓库设置中配置以下 Secrets：

```bash
# 必需的 Secrets
GITHUB_TOKEN          # GitHub Token (自动提供)

# 可选的 Secrets
DATABASE_PASSWORD     # 生产数据库密码
REDIS_PASSWORD        # 生产 Redis 密码
JWT_SECRET           # 生产 JWT Secret
GRAFANA_PASSWORD     # Grafana 管理密码
SLACK_WEBHOOK_URL    # 通知 Webhook
```

## 📊 监控和维护

### 监控面板

访问地址：
- **应用**: http://localhost:8080
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **API 文档**: http://localhost:8080/swagger/index.html

### 日志管理

#### 日志位置
```bash
# 应用日志
./logs/app.log

# Nginx 日志
./logs/nginx/access.log
./logs/nginx/error.log

# 数据库日志
docker logs go-progres-postgres
```

#### 日志轮转配置
```bash
# 创建 logrotate 配置
sudo vim /etc/logrotate.d/go-progres
```

```
/opt/go-progres/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 root root
    postrotate
        docker-compose -f /opt/go-progres/docker-compose.prod.yml restart nginx
    endscript
}
```

### 备份策略

#### 数据库备份
```bash
# 手动备份
docker-compose -f docker-compose.prod.yml exec postgres pg_dump -U postgres postgres > backup.sql

# 自动备份 (添加到 crontab)
0 2 * * * /opt/go-progres/scripts/backup.sh
```

#### 配置备份
```bash
# 备份配置文件
tar -czf config-backup-$(date +%Y%m%d).tar.gz .env nginx/ monitoring/
```

### 性能优化

#### 数据库优化
```sql
-- 创建索引
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_users_created_at ON users(created_at);

-- 分析表统计信息
ANALYZE users;
```

#### 应用优化
```bash
# 调整 Docker 资源限制
vim docker-compose.prod.yml
```

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
        reservations:
          cpus: '1.0'
          memory: 512M
```

## 🔧 故障排除

### 常见问题

#### 1. 服务无法启动
```bash
# 检查端口占用
sudo netstat -tlnp | grep :8080

# 检查 Docker 状态
sudo systemctl status docker

# 查看详细错误
docker-compose -f docker-compose.prod.yml logs app
```

#### 2. 数据库连接失败
```bash
# 检查数据库状态
docker-compose -f docker-compose.prod.yml exec postgres pg_isready

# 检查网络连接
docker network ls
docker network inspect go-progres_app-network
```

#### 3. 内存不足
```bash
# 检查系统资源
free -h
df -h

# 清理 Docker 资源
docker system prune -a
```

#### 4. 权限问题
```bash
# 修复文件权限
sudo chown -R $USER:$USER /opt/go-progres
sudo chmod +x scripts/*.sh
```

### 紧急恢复

#### 回滚到上一版本
```bash
# 停止当前服务
docker-compose -f docker-compose.prod.yml down

# 切换到上一版本
git checkout HEAD~1

# 重新部署
./scripts/deploy.sh production latest
```

#### 数据库恢复
```bash
# 恢复数据库备份
docker-compose -f docker-compose.prod.yml exec postgres psql -U postgres -d postgres < backup.sql
```

### 性能监控

#### 关键指标
- **响应时间**: < 100ms (P95)
- **错误率**: < 1%
- **CPU 使用率**: < 70%
- **内存使用率**: < 80%
- **磁盘使用率**: < 85%

#### 告警配置
在 Grafana 中配置告警规则：
1. 应用不可用
2. 响应时间过长
3. 错误率过高
4. 资源使用率过高

## 📞 支持

如果遇到问题，请：

1. 查看本文档的故障排除部分
2. 检查项目 Issues 页面
3. 提交新的 Issue 并包含：
   - 环境信息
   - 错误日志
   - 复现步骤

---

**最后更新**: 2026-02-11
**维护者**: 开发团队
**版本**: 1.0.0
