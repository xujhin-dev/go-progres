package otp

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type OTPService interface {
	Send(mobile string) (string, error)
	Verify(mobile, code string) bool
}

type otpService struct {
	rdb *redis.Client
}

func NewOTPService(rdb *redis.Client) OTPService {
	return &otpService{rdb: rdb}
}

// Send 生成并发送验证码
// 真实场景下应调用短信服务商接口 (如阿里云 SMS)
// 这里仅生成 6 位随机数并存入 Redis，同时打印到日志
func (s *otpService) Send(mobile string) (string, error) {
	// 1. 频率限制 (例如 1分钟内只能发一次)
	key := fmt.Sprintf("otp:%s", mobile)
	ttl, err := s.rdb.TTL(context.Background(), key).Result()
	if err == nil && ttl > 4*time.Minute { // 5分钟有效期，如果剩余 > 4分钟，说明刚发不久
		return "", fmt.Errorf("please wait before sending again")
	}

	// 2. 生成验证码
	// 为了演示方便，固定为 "123456"，或者使用 crypto/rand 生成
	code := "123456" 
	
	// 3. 存入 Redis (5分钟过期)
	if err := s.rdb.Set(context.Background(), key, code, 5*time.Minute).Err(); err != nil {
		return "", err
	}

	// 4. 发送 (Mock: 打印日志)
	log.Printf("[OTP] Sending code %s to %s", code, mobile)
	
	return code, nil
}

// Verify 验证验证码
// 验证成功后立即删除，防止重放
func (s *otpService) Verify(mobile, code string) bool {
	key := fmt.Sprintf("otp:%s", mobile)
	val, err := s.rdb.Get(context.Background(), key).Result()
	if err != nil {
		return false
	}

	if val == code {
		s.rdb.Del(context.Background(), key)
		return true
	}
	return false
}
