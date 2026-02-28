# Go Progress 项目结构

```
go-progres/
├── api/                    # API定义文件 (OpenAPI/Swagger)
├── assets/                  # 静态资源文件
├── build/                   # 构建输出文件
│   ├── coverage/            # 测试覆盖率报告
│   └── ...
├── cmd/                     # 应用程序入口点
│   ├── server/             # HTTP服务器
│   ├── worker/             # 后台任务处理器
│   └── cli/                # 命令行工具
├── configs/                  # 配置文件
├── deployments/              # 部署相关文件
├── docs/                    # 项目文档
│   ├── README.md            # 项目文档
│   ├── docs.go             # 文档生成代码
│   ├── swagger.json        # API规范
│   └── swagger.yaml        # API规范
├── internal/                 # 内部包
│   ├── domain/             # 业务领域
│   │   └── user/          # 用户领域
│   │       ├── model/      # 用户模型
│   │       ├── repository/ # 用户仓储
│   │       └── service/   # 用户服务
│   ├── infrastructure/     # 基础设施
│   └── application/       # 应用层
├── logs/                    # 日志文件
├── migrations/              # 数据库迁移
├── monitoring/              # 监控配置
├── nginx/                   # Nginx配置
├── pkg/                     # 公共包
│   ├── cache/              # 缓存实现
│   ├── database/           # 数据库工具
│   ├── utils/              # 工具函数
│   └── ...
├── scripts/                 # 脚本文件
├── temp/                    # 临时文件
├── tests/                   # 测试文件
├── tools/                   # 开发工具
├── .env                     # 环境变量
├── .env.example              # 环境变量示例
├── .gitignore                # Git忽略文件
├── Dockerfile               # Docker镜像构建
├── docker-compose.yml        # 开发环境Docker
├── docker-compose.prod.yml   # 生产环境Docker
├── go.mod                   # Go模块定义
├── go.sum                   # Go依赖锁定
├── Makefile                 # 构建脚本
├── sqlc.yaml                # SQLC配置
└── README.md                # 项目说明
```

## 目录说明

### 核心目录

- **`internal/`**: 内部业务逻辑，不对外暴露
  - `domain/`: 业务领域，包含模型、仓储、服务
  - `infrastructure/`: 基础设施实现
  - `application/`: 应用服务层

- **`pkg/`**: 可复用的公共包
  - `cache/`: 缓存服务实现
  - `database/`: 数据库连接和工具
  - `utils/`: 通用工具函数

- **`cmd/`**: 应用程序入口
  - `server/`: HTTP服务器启动
  - `worker/`: 后台任务
  - `cli/`: 命令行工具

### 配置和部署

- **`configs/`**: 配置文件模板
- **`deployments/`**: Kubernetes、Docker部署文件
- **`migrations/`**: 数据库版本迁移脚本
- **`monitoring/`**: 监控和日志配置

### 开发和工具

- **`scripts/`**: 自动化脚本
- **`tools/`**: 开发工具和脚本
- **`tests/`**: 集成测试和端到端测试
- **`docs/`**: API文档和架构文档

### 构建和输出

- **`build/`**: 构建输出
  - `coverage/`: 测试覆盖率报告
- **`logs/`**: 应用日志文件
- **`temp/`**: 临时文件

### 外部接口

- **`api/`**: API定义文件
- **`assets/`**: 静态资源

## 开发规范

1. **内部包**: `internal/`下的包不对外暴露
2. **公共包**: `pkg/`下的包可以对外使用
3. **领域驱动**: `internal/domain/`按业务领域组织
4. **依赖注入**: 使用依赖注入容器管理依赖
5. **测试覆盖**: 保持80%以上的测试覆盖率

## 构建命令

```bash
# 运行测试
make test

# 生成覆盖率报告
make test-coverage

# 构建应用
make build

# 运行服务
make run
```

## 部署

```bash
# 开发环境
docker-compose up -d

# 生产环境
docker-compose -f docker-compose.prod.yml up -d
```
