# 项目文档索引

欢迎查阅项目文档！本目录包含了项目的所有技术文档和指南。

## 📚 文档分类

### 🚀 快速开始

1. **[项目介绍](../README.md)**
   - 项目概述
   - 技术栈
   - 快速开始指南
   - API 接口文档

2. **[服务状态](SERVICE_STATUS.md)**
   - 当前服务状态
   - 可用接口列表
   - 配置说明
   - 测试指南

### 🏗️ 架构设计

1. **[架构优化详解](ARCHITECTURE_OPTIMIZATION.md)**
   - 模块自动注册机制
   - 架构设计思想
   - 技术实现细节
   - 优化前后对比

2. **[模块优化总结](MODULE_OPTIMIZATION_SUMMARY.md)**
   - 优化目标和成果
   - 核心组件说明
   - 使用指南
   - 最佳实践

### 👨‍💻 开发指南

1. **[添加新模块指南](ADD_NEW_MODULE.md)** ⭐
   - 详细步骤说明
   - 完整代码示例
   - 模块结构规范
   - 常见问题解答

2. **[修复总结](FIXES_SUMMARY.md)**
   - 已修复的问题列表
   - 新增功能说明
   - 数据库迁移指南
   - 测试建议

### 📊 项目管理

1. **[项目状态报告](PROJECT_STATUS_REPORT.md)**
   - 项目概览
   - 已完成工作
   - 性能指标
   - 后续规划

### 📖 API 文档

1. **[Swagger UI](http://localhost:8080/swagger/index.html)**
   - 在线 API 文档
   - 接口测试工具

2. **[Swagger JSON](http://localhost:8080/swagger/doc.json)**
   - API 规范文件

## 🎯 按角色查看

### 新手开发者

推荐阅读顺序：

1. [项目介绍](../README.md)
2. [服务状态](SERVICE_STATUS.md)
3. [添加新模块指南](ADD_NEW_MODULE.md)

### 架构师

推荐阅读顺序：

1. [架构优化详解](ARCHITECTURE_OPTIMIZATION.md)
2. [模块优化总结](MODULE_OPTIMIZATION_SUMMARY.md)
3. [项目状态报告](PROJECT_STATUS_REPORT.md)

### 项目经理

推荐阅读顺序：

1. [项目状态报告](PROJECT_STATUS_REPORT.md)
2. [修复总结](FIXES_SUMMARY.md)
3. [服务状态](SERVICE_STATUS.md)

## 🔧 工具和脚本

### 测试脚本

- **[API 测试脚本](../scripts/test_api.sh)**
  - 自动化 API 测试
  - 功能验证

### 数据库迁移

```bash
# 运行迁移
go run cmd/migrate/main.go

# 迁移文件位置
migrations/
```

## 📝 文档维护

### 文档更新规范

1. 所有文档使用 Markdown 格式
2. 文档标题使用中文
3. 代码示例使用英文注释
4. 保持文档结构清晰

### 文档目录结构

```
docs/
├── README.md                           # 本文档（索引）
├── ADD_NEW_MODULE.md                   # 添加新模块指南
├── ARCHITECTURE_OPTIMIZATION.md        # 架构优化详解
├── MODULE_OPTIMIZATION_SUMMARY.md      # 模块优化总结
├── FIXES_SUMMARY.md                    # 修复总结
├── SERVICE_STATUS.md                   # 服务状态
├── PROJECT_STATUS_REPORT.md            # 项目状态报告
├── docs.go                             # Swagger 文档生成
├── swagger.json                        # Swagger JSON
└── swagger.yaml                        # Swagger YAML
```

## 🔗 相关链接

### 在线资源

- [Swagger UI](http://localhost:8080/swagger/index.html)
- [健康检查](http://localhost:8080/health)
- [Prometheus 指标](http://localhost:8080/metrics)

### 外部文档

- [Gin 框架文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [Go 官方文档](https://go.dev/doc/)

## 💡 贡献指南

### 如何贡献文档

1. 发现文档问题或需要补充
2. 创建新的 Markdown 文件或修改现有文件
3. 更新本索引文件
4. 提交 Pull Request

### 文档规范

- 使用清晰的标题层级
- 提供代码示例
- 添加必要的图表
- 保持内容简洁明了

## 🆘 获取帮助

### 常见问题

1. **如何添加新模块？**
   - 查看 [添加新模块指南](ADD_NEW_MODULE.md)

2. **如何理解架构设计？**
   - 查看 [架构优化详解](ARCHITECTURE_OPTIMIZATION.md)

3. **如何查看 API 文档？**
   - 访问 [Swagger UI](http://localhost:8080/swagger/index.html)

4. **如何运行测试？**
   - 执行 `scripts/test_api.sh`

### 联系方式

- 项目仓库: [GitHub](https://github.com/your-repo)
- 问题反馈: [Issues](https://github.com/your-repo/issues)

---

**最后更新**: 2026-02-10
**维护者**: 开发团队
**版本**: 1.0.0
