# 如何添加新模块

本文档说明如何在项目中添加新的业务模块，无需修改 `main.go`。

## 🎯 设计理念

采用**模块自动注册机制**，每个模块通过 `init()` 函数自动注册到全局注册表，`main.go` 会自动发现并初始化所有模块。

## 📁 模块结构

每个模块遵循标准的 DDD 分层结构：

```
internal/domain/your_module/
├── handler/          # HTTP 处理层
│   └── your_handler.go
├── service/          # 业务逻辑层
│   └── your_service.go
├── repository/       # 数据访问层
│   └── your_repository.go
├── model/            # 数据模型
│   └── your_model.go
└── module.go         # 模块注册文件（核心）
```

## 🚀 添加新模块的步骤

### 步骤 1: 创建模块目录结构

```bash
mkdir -p internal/domain/product/{handler,service,repository,model}
```

### 步骤 2: 定义数据模型

创建 `internal/domain/product/model/product.go`:

```go
package model

import "gorm.io/gorm"

type Product struct {
    gorm.Model
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
}
```

### 步骤 3: 实现 Repository 层

创建 `internal/domain/product/repository/product_repository.go`:

```go
package repository

import (
    "user_crud_jwt/internal/domain/product/model"
    "gorm.io/gorm"
)

type ProductRepository interface {
    Create(product *model.Product) error
    GetByID(id uint) (*model.Product, error)
    GetList(offset, limit int) ([]model.Product, int64, error)
    Update(product *model.Product) error
    Delete(product *model.Product) error
}

type productRepository struct {
    db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
    return &productRepository{db: db}
}

func (r *productRepository) Create(product *model.Product) error {
    return r.db.Create(product).Error
}

func (r *productRepository) GetByID(id uint) (*model.Product, error) {
    var product model.Product
    if err := r.db.First(&product, id).Error; err != nil {
        return nil, err
    }
    return &product, nil
}

func (r *productRepository) GetList(offset, limit int) ([]model.Product, int64, error) {
    var products []model.Product
    var total int64

    if err := r.db.Model(&model.Product{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }

    if err := r.db.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
        return nil, 0, err
    }

    return products, total, nil
}

func (r *productRepository) Update(product *model.Product) error {
    return r.db.Save(product).Error
}

func (r *productRepository) Delete(product *model.Product) error {
    return r.db.Delete(product).Error
}
```

### 步骤 4: 实现 Service 层

创建 `internal/domain/product/service/product_service.go`:

```go
package service

import (
    "user_crud_jwt/internal/domain/product/model"
    "user_crud_jwt/internal/domain/product/repository"
)

type ProductService interface {
    CreateProduct(name, description string, price float64, stock int) (*model.Product, error)
    GetProduct(id uint) (*model.Product, error)
    GetProducts(page, limit int) ([]model.Product, int64, error)
    UpdateProduct(id uint, name, description string, price float64, stock int) (*model.Product, error)
    DeleteProduct(id uint) error
}

type productService struct {
    repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
    return &productService{repo: repo}
}

func (s *productService) CreateProduct(name, description string, price float64, stock int) (*model.Product, error) {
    product := &model.Product{
        Name:        name,
        Description: description,
        Price:       price,
        Stock:       stock,
    }

    if err := s.repo.Create(product); err != nil {
        return nil, err
    }

    return product, nil
}

func (s *productService) GetProduct(id uint) (*model.Product, error) {
    return s.repo.GetByID(id)
}

func (s *productService) GetProducts(page, limit int) ([]model.Product, int64, error) {
    if page <= 0 {
        page = 1
    }
    if limit <= 0 {
        limit = 10
    }
    offset := (page - 1) * limit
    return s.repo.GetList(offset, limit)
}

func (s *productService) UpdateProduct(id uint, name, description string, price float64, stock int) (*model.Product, error) {
    product, err := s.repo.GetByID(id)
    if err != nil {
        return nil, err
    }

    product.Name = name
    product.Description = description
    product.Price = price
    product.Stock = stock

    if err := s.repo.Update(product); err != nil {
        return nil, err
    }

    return product, nil
}

func (s *productService) DeleteProduct(id uint) error {
    product, err := s.repo.GetByID(id)
    if err != nil {
        return err
    }
    return s.repo.Delete(product)
}
```

### 步骤 5: 实现 Handler 层

创建 `internal/domain/product/handler/product_handler.go`:

```go
package handler

import (
    "net/http"
    "strconv"
    "user_crud_jwt/internal/domain/product/service"
    "user_crud_jwt/pkg/response"
    "user_crud_jwt/pkg/utils"

    "github.com/gin-gonic/gin"
)

type ProductHandler struct {
    service service.ProductService
}

func NewProductHandler(service service.ProductService) *ProductHandler {
    return &ProductHandler{service: service}
}

type CreateProductInput struct {
    Name        string  `json:"name" binding:"required"`
    Description string  `json:"description"`
    Price       float64 `json:"price" binding:"required,gt=0"`
    Stock       int     `json:"stock" binding:"required,gte=0"`
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
    var input CreateProductInput
    if err := c.ShouldBindJSON(&input); err != nil {
        response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
        return
    }

    product, err := h.service.CreateProduct(input.Name, input.Description, input.Price, input.Stock)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
        return
    }

    response.Success(c, product)
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
    var pagination utils.Pagination
    if err := c.ShouldBindQuery(&pagination); err != nil {
        response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
        return
    }

    products, total, err := h.service.GetProducts(pagination.Page, pagination.Limit)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
        return
    }

    result := utils.PageResult{
        List:  products,
        Total: total,
        Page:  pagination.Page,
        Limit: pagination.Limit,
    }
    response.Success(c, result)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid product ID")
        return
    }

    product, err := h.service.GetProduct(uint(id))
    if err != nil {
        response.Error(c, http.StatusNotFound, response.ErrServerInternal, "Product not found")
        return
    }

    response.Success(c, product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid product ID")
        return
    }

    var input CreateProductInput
    if err := c.ShouldBindJSON(&input); err != nil {
        response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
        return
    }

    product, err := h.service.UpdateProduct(uint(id), input.Name, input.Description, input.Price, input.Stock)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
        return
    }

    response.Success(c, product)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid product ID")
        return
    }

    if err := h.service.DeleteProduct(uint(id)); err != nil {
        response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
        return
    }

    response.Success(c, "Product deleted successfully")
}
```

### 步骤 6: 创建模块注册文件（核心）

创建 `internal/domain/product/module.go`:

```go
package product

import (
    "user_crud_jwt/internal/domain/product/handler"
    "user_crud_jwt/internal/domain/product/repository"
    "user_crud_jwt/internal/domain/product/service"
    "user_crud_jwt/internal/pkg/middleware"
    "user_crud_jwt/internal/pkg/registry"

    "github.com/gin-gonic/gin"
)

// ProductModule 产品模块
type ProductModule struct{}

func init() {
    // 自动注册模块 - 这是关键！
    registry.Register(&ProductModule{})
}

func (m *ProductModule) Name() string {
    return "product"
}

func (m *ProductModule) Priority() int {
    // 优先级：数字越小越先初始化
    // 1-9: 核心模块（如 user）
    // 10-99: 业务模块
    // 100+: 通用功能模块
    return 10
}

func (m *ProductModule) Init(ctx *registry.ModuleContext) error {
    // 1. 依赖注入
    productRepo := repository.NewProductRepository(ctx.DB)
    productService := service.NewProductService(productRepo)
    productHandler := handler.NewProductHandler(productService)

    // 2. 路由注册
    setupRoutes(ctx.Router, productHandler)

    return nil
}

func setupRoutes(r *gin.Engine, h *handler.ProductHandler) {
    g := r.Group("/products")

    // 公开路由
    g.GET("/", h.GetProducts)
    g.GET("/:id", h.GetProduct)

    // 需要认证的路由
    auth := g.Group("")
    auth.Use(middleware.AuthMiddleware())
    {
        auth.POST("/", h.CreateProduct)
        auth.PUT("/:id", h.UpdateProduct)
        auth.DELETE("/:id", h.DeleteProduct)
    }
}
```

### 步骤 7: 在 main.go 中导入模块

**只需添加一行导入！**

编辑 `cmd/server/main.go`:

```go
import (
    // 导入所有 domain 模块（触发 init 函数自动注册）
    _ "user_crud_jwt/internal/domain/common"
    _ "user_crud_jwt/internal/domain/coupon"
    _ "user_crud_jwt/internal/domain/moment"
    _ "user_crud_jwt/internal/domain/payment"
    _ "user_crud_jwt/internal/domain/product"  // 👈 只需添加这一行！
    _ "user_crud_jwt/internal/domain/user"

    // ... 其他导入
)
```

### 步骤 8: 创建数据库迁移

创建 `migrations/000005_add_products_table.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_deleted_at ON products(deleted_at);
```

创建 `migrations/000005_add_products_table.down.sql`:

```sql
DROP TABLE IF EXISTS products;
```

### 步骤 9: 运行迁移并启动服务

```bash
# 运行数据库迁移
go run cmd/migrate/main.go

# 启动服务（会自动发现并初始化新模块）
go run cmd/server/main.go
```

## ✅ 完成！

新模块已经自动注册并可用，无需修改 `main.go` 的任何逻辑代码！

### 测试新模块

```bash
# 创建产品
curl -X POST http://localhost:8080/products/ \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "iPhone 15",
    "description": "Latest iPhone",
    "price": 999.99,
    "stock": 100
  }'

# 获取产品列表
curl http://localhost:8080/products/?page=1&limit=10

# 获取单个产品
curl http://localhost:8080/products/1
```

## 🎨 模块优先级说明

模块的 `Priority()` 方法决定初始化顺序：

- **1-9**: 核心基础模块（如 user，其他模块可能依赖它）
- **10-99**: 普通业务模块（如 product, order, coupon）
- **100+**: 通用功能模块（如 common, upload）

如果模块 A 依赖模块 B，确保 B 的优先级数字小于 A。

## 🔧 高级用法

### 模块间依赖

如果新模块依赖其他模块的服务：

```go
func (m *OrderModule) Init(ctx *registry.ModuleContext) error {
    // 获取其他模块的服务
    userRepo := userRepo.NewUserRepository(ctx.DB)
    userService := userService.NewUserService(userRepo)

    productRepo := productRepo.NewProductRepository(ctx.DB)
    productService := productService.NewProductService(productRepo)

    // 注入到当前模块
    orderRepo := repository.NewOrderRepository(ctx.DB)
    orderService := service.NewOrderService(orderRepo, userService, productService)
    orderHandler := handler.NewOrderHandler(orderService)

    setupRoutes(ctx.Router, orderHandler)
    return nil
}
```

### 条件注册

如果模块需要特定配置才能启用：

```go
func (m *PaymentModule) Init(ctx *registry.ModuleContext) error {
    // 检查配置
    if config.GlobalConfig.Payment.Enabled {
        // 初始化模块
        // ...
    } else {
        logger.Log.Info("Payment module is disabled")
        return nil
    }

    return nil
}
```

## 📊 优势总结

✅ **零侵入**: 添加新模块无需修改 `main.go` 的业务逻辑
✅ **自动发现**: 通过 `init()` 函数自动注册
✅ **解耦合**: 每个模块完全独立，可以单独开发和测试
✅ **易维护**: 模块结构清晰，职责明确
✅ **可扩展**: 轻松添加新功能，不影响现有代码
✅ **优先级控制**: 灵活控制模块初始化顺序

## 🚫 注意事项

1. **包名冲突**: 确保新模块的包名唯一
2. **路由冲突**: 检查路由路径不与现有模块冲突
3. **数据库迁移**: 记得创建和运行迁移文件
4. **依赖顺序**: 如果有依赖关系，设置正确的优先级
5. **错误处理**: `Init()` 方法返回错误会导致服务启动失败

---

**Happy Coding!** 🎉
