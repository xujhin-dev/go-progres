# Golang Commercial-Grade REST API Template

这是一个基于 **Gin**, **GORM (Postgres)**, **JWT** 的现代化 Go 后端项目模板。采用了 **DDD (领域驱动设计)** 和 **Clean Architecture (整洁架构)** 的思想进行分层，专为可扩展的商用项目设计。

## 🚀 项目特性

- **模块化架构**：基于 DDD 的目录结构，业务模块（如 `user`）高度内聚，易于扩展新业务（如 `mall`, `order`）。
- **模块自动注册**：添加新模块只需一行代码，无需修改 main.go 的业务逻辑。
- **RESTful API**：标准化的 HTTP 接口设计，JSON 响应使用驼峰命名。
- **JWT 认证**：基于 `golang-jwt` 的无状态认证机制，集成中间件保护路由。
- **配置管理**：使用 `Viper` 支持多环境配置 (YAML + 环境变量)。
- **ORM 集成**：使用 `GORM` 进行数据库操作，支持自动迁移。
- **依赖注入**：在模块层自动组装依赖，清晰且易于测试。
- **完整监控**：集成 Prometheus 指标、请求追踪、健康检查。

## 📚 文档

- **[完整文档索引](docs/README.md)** - 所有文档的导航入口
- **[添加新模块指南](docs/ADD_NEW_MODULE.md)** - 如何快速添加新业务模块
- **[架构优化详解](docs/ARCHITECTURE_OPTIMIZATION.md)** - 模块自动注册机制详解
- **[服务状态](docs/SERVICE_STATUS.md)** - 当前服务状态和使用指南
- **[API 文档](http://localhost:8080/swagger/index.html)** - Swagger 在线文档

## 📂 目录结构

```
├── cmd
│   └── server
│       └── main.go           # 🚀 项目入口，负责依赖组装和服务器启动
├── configs
│   └── config.yaml           # ⚙️ 全局配置文件
├── internal
│   ├── domain                # 📦 业务领域 (按模块划分)
│   │   └── user              # [示例] 用户模块
│   │       ├── handler       # HTTP 处理层 (Controller)
│   │       ├── service       # 业务逻辑层 (Business Logic)
│   │       ├── repository    # 数据访问层 (DAO)
│   │       ├── model         # 数据模型 (Entity)
│   │       └── router.go     # 模块路由注册入口
│   └── pkg                   # 🔒 内部共享组件 (仅限项目内部使用)
│       ├── config            # 配置加载逻辑
│       └── middleware        # 全局中间件 (如 Auth)
├── pkg                       # 🌐 公共库 (可被外部引用)
│   ├── database              # 数据库连接与初始化
│   └── utils                 # 通用工具 (如 JWT 生成)
├── go.mod                    # 依赖管理
└── README.md                 # 项目文档
```

## 🛠️ 快速开始

### 1. 环境准备

- **Go**: 1.18+
- **PostgreSQL**: 12+

### 2. 配置数据库

确保 PostgreSQL 正在运行，并创建一个数据库（默认配置名为 `postgres`）。
修改 `configs/config.yaml` 适配你的本地环境：

```yaml
database:
  host: "localhost"
  port: "5432"
  user: "your_user"
  password: "your_password"
  dbname: "your_dbname"
```

### 3. 运行项目

```bash
# 下载依赖
go mod tidy

# 运行服务器
go run cmd/server/main.go
```

服务器默认启动在 `http://localhost:8080`。

## 🔌 API 接口文档

### 认证模块 (Auth)

| 方法 | 路径             | 描述                  |
| :--- | :--------------- | :-------------------- |
| POST | `/auth/register` | 用户注册              |
| POST | `/auth/login`    | 用户登录 (返回 Token) |

**请求示例 (注册):**

```json
{
  "username": "testuser",
  "password": "password123",
  "email": "test@example.com"
}
```

### 用户模块 (User) - 需携带 Bearer Token

| 方法   | 路径         | 描述         |
| :----- | :----------- | :----------- |
| GET    | `/users/`    | 获取所有用户 |
| GET    | `/users/:id` | 获取特定用户 |
| PUT    | `/users/:id` | 更新用户信息 |
| DELETE | `/users/:id` | 删除用户     |

**Header:**

```
Authorization: Bearer <your_token_here>
```

## 🏗️ 如何扩展新模块 (例如：商城模块)

1.  在 `internal/domain/` 下创建 `mall` 目录。
2.  建立标准分层目录：`handler`, `service`, `repository`, `model`。
3.  实现业务逻辑。
4.  创建 `internal/domain/mall/router.go` 暴露路由注册函数 `SetupMallRoutes`。
5.  在 `cmd/server/main.go` 中调用 `SetupMallRoutes`。

---

**Happy Coding!** 🚀
