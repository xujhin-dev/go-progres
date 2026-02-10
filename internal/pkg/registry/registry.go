package registry

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/redis/go-redis/v9"
)

// ModuleContext 模块初始化所需的上下文
type ModuleContext struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Router *gin.Engine
}

// Module 模块接口
type Module interface {
	// Name 返回模块名称
	Name() string

	// Init 初始化模块（依赖注入、路由注册等）
	Init(ctx *ModuleContext) error

	// Priority 返回初始化优先级（数字越小越先初始化）
	// 例如：user 模块可能需要先于 payment 模块初始化
	Priority() int
}

// moduleRegistry 全局模块注册表
var moduleRegistry = make(map[string]Module)

// Register 注册模块
func Register(module Module) {
	moduleRegistry[module.Name()] = module
}

// GetModules 获取所有已注册的模块
func GetModules() map[string]Module {
	return moduleRegistry
}

// InitModules 按优先级初始化所有模块
func InitModules(ctx *ModuleContext) error {
	// 按优先级排序
	modules := make([]Module, 0, len(moduleRegistry))
	for _, m := range moduleRegistry {
		modules = append(modules, m)
	}

	// 简单的冒泡排序（模块数量不多，性能足够）
	for i := 0; i < len(modules); i++ {
		for j := i + 1; j < len(modules); j++ {
			if modules[i].Priority() > modules[j].Priority() {
				modules[i], modules[j] = modules[j], modules[i]
			}
		}
	}

	// 按顺序初始化
	for _, module := range modules {
		if err := module.Init(ctx); err != nil {
			return err
		}
	}

	return nil
}
