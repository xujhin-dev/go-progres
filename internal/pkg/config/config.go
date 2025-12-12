package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
	SSLMode  string
	TimeZone string
}

type JWTConfig struct {
	Secret string
	Expire int64 // 小时
}

var GlobalConfig Config

// LoadConfig 加载配置
func LoadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("jwt.expire", 24)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Config file not found, using defaults or env vars: %v", err)
	}

	// 绑定环境变量
	viper.AutomaticEnv()

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	// 手动覆盖，以防 viper 无法正确解析复杂结构或环境变量
	if host := os.Getenv("DB_HOST"); host != "" {
		GlobalConfig.Database.Host = host
	}
	// ... 其他环境变量映射
}
