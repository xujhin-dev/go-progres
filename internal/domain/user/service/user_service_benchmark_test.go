package service

import (
	"context"
	"fmt"
	"testing"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
)

// SimpleOTPService 简单的OTP服务用于基准测试
type SimpleOTPService struct{}

func (s *SimpleOTPService) Send(mobile string) (string, error) {
	return "123456", nil
}

func (s *SimpleOTPService) Verify(mobile, code string) bool {
	return code == "123456"
}

// BenchmarkUserService_LoginOrRegister 登录注册性能测试
func BenchmarkUserService_LoginOrRegister(b *testing.B) {
	// 使用内存仓库进行性能测试
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.LoginOrRegister(ctx, "13800138000", "123456")
	}
	b.StopTimer()
}

// BenchmarkUserService_GetUsers 用户列表性能测试
func BenchmarkUserService_GetUsers(b *testing.B) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	// 预先创建一些用户
	for i := 0; i < 100; i++ {
		user := &model.User{
			ID:       fmt.Sprintf("user-%d", i),
			Mobile:   fmt.Sprintf("138001380%02d", i),
			Nickname: fmt.Sprintf("User-%d", i),
			Role:     model.RoleUser,
			Status:   model.StatusNormal,
		}
		mockRepo.Create(ctx, user)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = service.GetUsers(ctx, 1, 10)
	}
	b.StopTimer()
}

// BenchmarkUserService_GetUser 单个用户查询性能测试
func BenchmarkUserService_GetUser(b *testing.B) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		ID:       "benchmark-user",
		Mobile:   "13800138000",
		Nickname: "Benchmark User",
		Role:     model.RoleUser,
		Status:   model.StatusNormal,
	}
	mockRepo.Create(ctx, user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetUser(ctx, "benchmark-user")
	}
	b.StopTimer()
}
