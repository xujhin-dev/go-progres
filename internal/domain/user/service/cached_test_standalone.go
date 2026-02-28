package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOTPService 简单的OTP服务用于测试
type TestOTPService struct{}

func (s *TestOTPService) Send(mobile string) (string, error) {
	return "123456", nil
}

func (s *TestOTPService) Verify(mobile, code string) bool {
	return code == "123456"
}

// TestCacheService 模拟缓存服务
type TestCacheService struct {
	data map[string]*cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

func NewTestCacheService() *TestCacheService {
	return &TestCacheService{
		data: make(map[string]*cacheItem),
	}
}

var ErrCacheMiss = errors.New("cache miss")

func (m *TestCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.data[key]
	if !exists || time.Now().After(item.expiration) {
		return ErrCacheMiss
	}

	data, err := json.Marshal(item.value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	return json.Unmarshal(data, dest)
}

func (m *TestCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	return nil
}

func (m *TestCacheService) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

func (m *TestCacheService) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.data[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(item.expiration) {
		delete(m.data, key)
		return false, nil
	}

	return true, nil
}

func (m *TestCacheService) GetWithTTL(ctx context.Context, key string, dest interface{}) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.data[key]
	if !exists {
		return 0, ErrCacheMiss
	}

	if time.Now().After(item.expiration) {
		delete(m.data, key)
		return 0, ErrCacheMiss
	}

	ttl := time.Until(item.expiration)

	data, err := json.Marshal(item.value)
	if err != nil {
		return 0, fmt.Errorf("cache marshal error: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return 0, fmt.Errorf("cache unmarshal error: %w", err)
	}

	return ttl, nil
}

func (m *TestCacheService) SetWithTTL(ctx context.Context, key string, value interface{}) error {
	return m.Set(ctx, key, value, time.Hour)
}

func (m *TestCacheService) InvalidatePattern(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key := range m.data {
		if matched, _ := filepath.Match(pattern, key); matched {
			delete(m.data, key)
		}
	}

	return nil
}

func (m *TestCacheService) GetMultiple(ctx context.Context, keys []string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]interface{}, len(keys))
	for i, key := range keys {
		if item, exists := m.data[key]; exists && !time.Now().After(item.expiration) {
			results[i] = item.value
		}
	}

	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	return json.Unmarshal(data, dest)
}

func (m *TestCacheService) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]*cacheItem)
	return nil
}

func TestCachedUserService_BasicOperations(t *testing.T) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &TestOTPService{}
	mockCache := NewTestCacheService()
	service := NewCachedUserService(mockRepo, mockOTP, mockCache)
	ctx := context.Background()

	t.Run("New user registration with cache", func(t *testing.T) {
		mobile := "13800138000"
		code := "123456"

		// 发送OTP
		err := service.SendOTP(ctx, mobile)
		assert.NoError(t, err)

		// 注册新用户
		token, err := service.LoginOrRegister(ctx, mobile, code)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// 验证用户已缓存
		cacheKey := "user:mobile:" + mobile
		var cachedUser model.User
		err = mockCache.Get(ctx, cacheKey, &cachedUser)
		assert.NoError(t, err)
		assert.Equal(t, mobile, cachedUser.Mobile)
	})

	t.Run("Get user from cache", func(t *testing.T) {
		mobile := "13800138001"
		code := "123456"

		// 注册用户
		err := service.SendOTP(ctx, mobile)
		assert.NoError(t, err)

		token, err := service.LoginOrRegister(ctx, mobile, code)
		assert.NoError(t, err)

		// 获取用户信息
		user, err := service.GetUser(ctx, token)
		assert.NoError(t, err)
		assert.Equal(t, mobile, user.Mobile)

		// 验证用户被缓存
		cacheKey := "user:id:" + user.ID
		var cachedUser model.User
		err = mockCache.Get(ctx, cacheKey, &cachedUser)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, cachedUser.ID)
	})

	t.Run("Update user and cache", func(t *testing.T) {
		mobile := "13800138002"
		code := "123456"

		// 注册用户
		err := service.SendOTP(ctx, mobile)
		assert.NoError(t, err)

		token, err := service.LoginOrRegister(ctx, mobile, code)
		require.NoError(t, err)

		// 获取用户ID
		user, err := service.GetUser(ctx, token)
		require.NoError(t, err)

		// 更新用户信息
		newNickname := "Updated User"
		newAvatarURL := "https://example.com/avatar.jpg"

		updatedUser, err := service.UpdateUser(ctx, user.ID, newNickname, newAvatarURL)
		assert.NoError(t, err)
		assert.Equal(t, newNickname, updatedUser.Nickname)
		assert.Equal(t, newAvatarURL, updatedUser.AvatarURL)

		// 验证缓存被更新
		cacheKey := "user:id:" + user.ID
		var cachedUser model.User
		err = mockCache.Get(ctx, cacheKey, &cachedUser)
		assert.NoError(t, err)

		assert.Equal(t, newNickname, cachedUser.Nickname)
		assert.Equal(t, newAvatarURL, cachedUser.AvatarURL)
	})

	t.Run("Delete user and clear cache", func(t *testing.T) {
		mobile := "13800138003"
		code := "123456"

		// 注册用户
		err := service.SendOTP(ctx, mobile)
		assert.NoError(t, err)

		token, err := service.LoginOrRegister(ctx, mobile, code)
		require.NoError(t, err)

		// 获取用户ID
		user, err := service.GetUser(ctx, token)
		require.NoError(t, err)

		// 删除用户
		err = service.DeleteUser(ctx, user.ID)
		assert.NoError(t, err)

		// 验证缓存被清除
		cacheKey := "user:id:" + user.ID
		var cachedUser model.User
		err = mockCache.Get(ctx, cacheKey, &cachedUser)
		assert.Error(t, err)
		assert.Equal(t, ErrCacheMiss, err)
	})
}

func TestCachedUserService_Performance(t *testing.T) {
	mockRepo := repository.NewSimpleUserRepository(nil)
	mockOTP := &TestOTPService{}
	mockCache := NewTestCacheService()
	service := NewCachedUserService(mockRepo, mockOTP, mockCache)
	ctx := context.Background()

	// 创建用户
	mobile := "13800138004"
	code := "123456"

	err := service.SendOTP(ctx, mobile)
	assert.NoError(t, err)

	token, err := service.LoginOrRegister(ctx, mobile, code)
	require.NoError(t, err)

	// 获取用户ID
	_, err = service.GetUser(ctx, token)
	require.NoError(t, err)

	// 测试缓存命中
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_, err := service.GetUser(ctx, token)
		assert.NoError(t, err)
	}
	duration := time.Since(start)

	t.Logf("1000 cache hits took: %v (avg: %v per request)", duration, duration/1000)
	assert.Less(t, duration, 100*time.Millisecond, "Cache should be fast")
}
