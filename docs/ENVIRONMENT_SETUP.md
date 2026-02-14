# 环境配置说明

## 支持的环境

本项目支持以下环境配置：

### 1. 开发环境 (dev)
- **配置文件**: `config.dev.yaml`
- **环境变量**: `APP_ENV=dev`
- **数据库**: `postgres_dev`
- **Redis DB**: 0
- **特殊验证码**: 支持 (123456)
- **调试模式**: 开启

### 2. 测试环境 (test)
- **配置文件**: `config.test.yaml`
- **环境变量**: `APP_ENV=test`
- **数据库**: `postgres_test`
- **Redis DB**: 1
- **特殊验证码**: 支持 (123456)
- **调试模式**: 开启

### 3. 生产环境 (prod)
- **配置文件**: `config.prod.yaml`
- **环境变量**: `APP_ENV=prod`
- **数据库**: `postgres_prod`
- **Redis DB**: 0
- **特殊验证码**: 不支持
- **调试模式**: 关闭

### 4. 本地测试环境 (local-test)
- **配置文件**: `config.local-test.yaml`
- **环境变量**: `APP_ENV=local-test`
- **数据库**: `postgres_test`
- **Redis DB**: 1
- **特殊验证码**: 支持 (123456)
- **调试模式**: 开启

### 5. 本地生产环境 (local-prod)
- **配置文件**: `config.local-prod.yaml`
- **环境变量**: `APP_ENV=local-prod`
- **数据库**: `postgres_prod`
- **Redis DB**: 0
- **特殊验证码**: 不支持
- **调试模式**: 开启

## 使用方法

### 1. 设置环境变量
```bash
# Linux/macOS
export APP_ENV=test

# Windows
set APP_ENV=test
```

### 2. 运行应用
```bash
go run cmd/server/main.go
```

应用会自动根据 `APP_ENV` 环境变量加载对应的配置文件。

## 特殊验证码

在以下环境中可以使用特殊验证码 `123456` 进行登录，无需发送短信：
- dev (开发环境)
- test (测试环境)  
- local-test (本地测试环境)

这样可以方便开发和测试，避免频繁发送短信验证码。

## 配置优先级

1. 环境变量 (最高优先级)
2. 配置文件
3. 默认值 (最低优先级)

## 数据库迁移

新增了 token 相关字段，需要运行数据库迁移：

```bash
# 如果使用 golang-migrate
migrate -path ./migrations -database "postgres://user:password@localhost/dbname?sslmode=disable" up

# 或者应用启动时自动迁移（如果配置了自动迁移）
```

## 安全注意事项

1. 生产环境必须使用强密码的 JWT Secret
2. 生产环境不支持特殊验证码
3. 生产环境建议启用 SSL
4. 不要在代码中硬编码敏感信息
