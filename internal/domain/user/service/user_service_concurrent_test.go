package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
	"user_crud_jwt/internal/domain/user/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentLoginOrRegister 并发登录注册测试
func TestConcurrentLoginOrRegister(t *testing.T) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	const numGoroutines = 100
	const numRequests = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numRequests)

	// 启动多个goroutine并发登录
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numRequests; j++ {
				mobile := fmt.Sprintf("13800138%03d", workerID)
				token, err := service.LoginOrRegister(ctx, mobile, "123456")
				if err != nil {
					errors <- fmt.Errorf("worker %d, request %d: %v", workerID, j, err)
					continue
				}
				if token == "" {
					errors <- fmt.Errorf("worker %d, request %d: empty token", workerID, j)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("Concurrent login error: %v", err)
	}

	// 验证用户数量
	users, total, err := service.GetUsers(ctx, 1, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines), total)
	assert.Len(t, users, numGoroutines)
}

// TestConcurrentUserOperations 并发用户操作测试
func TestConcurrentUserOperations(t *testing.T) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	// 预先创建一些用户
	userIDs := make([]string, 50)
	for i := 0; i < 50; i++ {
		mobile := fmt.Sprintf("13800138%03d", i)
		token, err := service.LoginOrRegister(ctx, mobile, "123456")
		require.NoError(t, err)
		require.NotEmpty(t, token)

		users, _, err := service.GetUsers(ctx, 1, 100)
		require.NoError(t, err)
		require.Len(t, users, i+1)
		userIDs[i] = users[i].ID
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// 并发执行各种用户操作
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			switch workerID % 4 {
			case 0: // 查询用户
				for j := 0; j < 10; j++ {
					userID := userIDs[j%len(userIDs)]
					_, err := service.GetUser(ctx, userID)
					if err != nil {
						errors <- fmt.Errorf("worker %d get user %d: %v", workerID, j, err)
						return
					}
				}
			case 1: // 更新用户
				for j := 0; j < 5; j++ {
					userID := userIDs[j%len(userIDs)]
					nickname := fmt.Sprintf("Updated-%d-%d", workerID, j)
					_, err := service.UpdateUser(ctx, userID, nickname, "")
					if err != nil {
						errors <- fmt.Errorf("worker %d update user %d: %v", workerID, j, err)
						return
					}
				}
			case 2: // 升级会员
				for j := 0; j < 3; j++ {
					userID := userIDs[j%len(userIDs)]
					err := service.UpgradeMember(ctx, userID, 30*24*time.Hour)
					if err != nil {
						errors <- fmt.Errorf("worker %d upgrade member %d: %v", workerID, j, err)
						return
					}
				}
			case 3: // 查询用户列表
				for j := 0; j < 20; j++ {
					_, _, err := service.GetUsers(ctx, 1, 10)
					if err != nil {
						errors <- fmt.Errorf("worker %d get users %d: %v", workerID, j, err)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// TestConcurrentSameUser 并发操作同一用户测试
func TestConcurrentSameUser(t *testing.T) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	mobile := "13800138000"

	// 先创建用户
	_, err := service.LoginOrRegister(ctx, mobile, "123456")
	require.NoError(t, err)

	// 获取用户ID
	users, _, err := service.GetUsers(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, users, 1)
	userID := users[0].ID

	const numGoroutines = 20
	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	// 并发更新同一用户
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			nickname := fmt.Sprintf("Concurrent-%d", workerID)
			_, err := service.UpdateUser(ctx, userID, nickname, "")
			if err == nil {
				successCount <- 1
			}
		}(i)
	}

	wg.Wait()
	close(successCount)

	// 统计成功的操作数
	successes := 0
	for range successCount {
		successes++
	}

	// 至少应该有一些操作成功
	assert.Greater(t, successes, 0, "At least some concurrent updates should succeed")
	t.Logf("Concurrent updates: %d/%d succeeded", successes, numGoroutines)
}

// TestRaceConditionDetection 竞态条件检测测试
func TestRaceConditionDetection(t *testing.T) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &SimpleOTPService{}
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	const numGoroutines = 50
	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	// 并发注册相同手机号的用户
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			mobile := "13800138001" // 所有goroutine使用相同手机号
			_, err := service.LoginOrRegister(ctx, mobile, "123456")
			if err != nil {
				t.Logf("Worker %d failed: %v", workerID, err)
				return
			}

			// 成功注册/登录
			successCount <- 1
		}(i)
	}

	wg.Wait()
	close(successCount)

	// 统计成功的操作数
	successes := 0
	for range successCount {
		successes++
	}

	// 验证只有一个用户被创建
	users, total, err := service.GetUsers(ctx, 1, 1000) // 使用足够大的limit
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "Should only create one user for same mobile")
	assert.Len(t, users, 1, "Should only have one user")
	assert.Equal(t, "13800138001", users[0].Mobile, "Mobile should match")

	// 所有成功的操作都应该是对同一个用户的操作
	assert.Equal(t, successes, numGoroutines, "All operations should succeed, but only one user created")

	t.Logf("Race condition test: %d goroutines competed for same mobile, created 1 user, %d successful operations", numGoroutines, successes)
}
