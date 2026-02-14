package config

import (
	"errors"
	"log"
	"os"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	App      AppConfig      `mapstructure:"app"`
	OSS      OSSConfig      `mapstructure:"oss"`
	Push     PushConfig     `mapstructure:"push"`
	Alipay   AlipayConfig   `mapstructure:"alipay"`
	Wechat   WechatPayConfig `mapstructure:"wechat"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Port     string `mapstructure:"port"`
	SSLMode  string `mapstructure:"sslmode"`
	TimeZone string `mapstructure:"timezone"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int64  `mapstructure:"expire"` // 小时
}

type AppConfig struct {
	Env         string `mapstructure:"env"`
	Debug       bool   `mapstructure:"debug"`
	TestOTPCode string `mapstructure:"test_otp_code"`
}

type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
}

type PushConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	AppKey          int64  `mapstructure:"app_key"`
	RegionID        string `mapstructure:"region_id"` // e.g., "cn-hangzhou"
}

type AlipayConfig struct {
	AppID        string `mapstructure:"app_id"`
	PrivateKey   string `mapstructure:"private_key"`   // 应用私钥
	PublicKey    string `mapstructure:"public_key"`    // 支付宝公钥 (不是应用公钥)
	NotifyURL    string `mapstructure:"notify_url"`    // 异步通知地址
	ReturnURL    string `mapstructure:"return_url"`    // 同步跳转地址
	IsProduction bool   `mapstructure:"is_production"` // 是否生产环境
}

type WechatPayConfig struct {
	AppID           string `mapstructure:"app_id"`
	MchID           string `mapstructure:"mch_id"`
	MchCertificateSerial string `mapstructure:"mch_cert_serial"`
	MchPrivateKey   string `mapstructure:"mch_private_key"`
	APIv3Key        string `mapstructure:"apiv3_key"`
	NotifyURL       string `mapstructure:"notify_url"`
}

var GlobalConfig Config

// Validate 验证配置
func (c *Config) Validate() error {
	// JWT 配置验证
	if c.JWT.Secret == "" || c.JWT.Secret == "your_super_secret_key" {
		return errors.New("please set a secure JWT secret in production")
	}
	if len(c.JWT.Secret) < 32 {
		return errors.New("JWT secret should be at least 32 characters")
	}

	// 数据库配置验证
	if c.Database.Host == "" || c.Database.User == "" || c.Database.DBName == "" {
		return errors.New("database configuration is incomplete")
	}

	// Redis 配置验证
	if c.Redis.Addr == "" {
		return errors.New("redis address is required")
	}

	return nil
}

// LoadConfig 加载配置
func LoadConfig() {
	// 获取环境变量，默认为dev
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	// 根据环境选择配置文件
	configName := "config"
	if env != "" && env != "dev" {
		configName = "config." + env
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("jwt.expire", 24)
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("app.env", "dev")
	viper.SetDefault("app.debug", true)

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
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		GlobalConfig.Redis.Addr = redisAddr
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		GlobalConfig.JWT.Secret = jwtSecret
	}

	// 验证配置
	if err := GlobalConfig.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Configuration loaded and validated successfully. Environment: %s", GlobalConfig.App.Env)
}
