package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/internal/pkg/otp"
	"user_crud_jwt/pkg/cache"
	"user_crud_jwt/pkg/utils"

	"gorm.io/gorm"
)

// CachedUserService 带缓存的用户服务
type CachedUserService struct {
	repo  repository.UserRepository
	otp   otp.OTPService
	cache cache.CacheService
}

// NewCachedUserService 创建带缓存的用户服务
func NewCachedUserService(repo repository.UserRepository, otp otp.OTPService, cache cache.CacheService) UserService {
	return &CachedUserService{
		repo:  repo,
		otp:   otp,
		cache: cache,
	}
}

// 缓存键常量
const (
	UserCacheKeyPrefix     = "user:"
	UserListCacheKeyPrefix = "user_list:"
	UserCacheTTL           = time.Hour * 2
	UserListCacheTTL       = time.Minute * 30
)

// getUserCacheKey 获取用户缓存键
func (s *CachedUserService) getUserCacheKey(id string) string {
	return fmt.Sprintf("%s%s", UserCacheKeyPrefix, id)
}

// getUserListCacheKey 获取用户列表缓存键
func (s *CachedUserService) getUserListCacheKey(page, limit int) string {
	return fmt.Sprintf("%s%d:%d", UserListCacheKeyPrefix, page, limit)
}

// invalidateUserCache 清除用户相关缓存
func (s *CachedUserService) invalidateUserCache(ctx context.Context, userID string) error {
	// 清除用户缓存
	if err := s.cache.Delete(ctx, s.getUserCacheKey(userID)); err != nil {
		return fmt.Errorf("failed to invalidate user cache: %w", err)
	}

	// 清除用户列表缓存（所有页）
	pattern := UserListCacheKeyPrefix + "*"
	if err := s.cache.InvalidatePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate user list cache: %w", err)
	}

	return nil
}

// LoginOrRegister 登录或注册
func (s *CachedUserService) LoginOrRegister(mobile, code string) (string, error) {
	// 1. 验证验证码
	if !s.otp.Verify(mobile, code) {
		return "", errors.New("invalid verification code")
	}

	// 2. 查询用户是否存在
	user, err := s.repo.GetByMobile(mobile)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 3. 不存在则注册
			user = &model.User{
				Mobile:   mobile,
				Nickname: "User_" + mobile[len(mobile)-4:], // 默认昵称
				Role:     model.RoleUser,
			}
			if err := s.repo.Create(user); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	// 4. 检查用户状态
	if user.Status == model.StatusBanned {
		if user.BannedUntil != nil && time.Now().After(*user.BannedUntil) {
			user.Status = model.StatusNormal
			user.BannedUntil = nil
			s.repo.Update(user)
		} else {
			return "", errors.New("account is banned")
		}
	}
	if user.Status == model.StatusDeleted {
		return "", errors.New("account has been deleted")
	}

	// 5. 生成 Token
	return utils.GenerateToken(user.ID, user.Role)
}

func (s *CachedUserService) SendOTP(mobile string) error {
	_, err := s.otp.Send(mobile)
	return err
}

// GetUsers 获取用户列表（带缓存）
func (s *CachedUserService) GetUsers(page, limit int) ([]model.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	ctx := context.Background()
	cacheKey := s.getUserListCacheKey(page, limit)

	// 尝试从缓存获取
	var cachedResult struct {
		Users []model.User `json:"users"`
		Total int64        `json:"total"`
	}

	if err := s.cache.Get(ctx, cacheKey, &cachedResult); err == nil {
		return cachedResult.Users, cachedResult.Total, nil
	}

	// 缓存未命中，从数据库获取
	offset := (page - 1) * limit
	users, total, err := s.repo.GetList(offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// 缓存结果
	cachedResult.Users = users
	cachedResult.Total = total
	if err := s.cache.Set(ctx, cacheKey, cachedResult, UserListCacheTTL); err != nil {
		// 缓存失败不影响业务逻辑，只记录日志
		fmt.Printf("Warning: failed to cache user list: %v\n", err)
	}

	return users, total, nil
}

// GetUser 获取单个用户（带缓存）
func (s *CachedUserService) GetUser(id string) (*model.User, error) {
	ctx := context.Background()
	cacheKey := s.getUserCacheKey(id)

	// 尝试从缓存获取
	var user model.User
	if err := s.cache.Get(ctx, cacheKey, &user); err == nil {
		return &user, nil
	}

	// 缓存未命中，从数据库获取
	userData, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	if err := s.cache.Set(ctx, cacheKey, userData, UserCacheTTL); err != nil {
		fmt.Printf("Warning: failed to cache user: %v\n", err)
	}

	return userData, nil
}

// UpdateUser 更新用户（带缓存失效）
func (s *CachedUserService) UpdateUser(id string, nickname, avatarURL string) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user.Nickname = nickname
	user.AvatarURL = avatarURL

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	// 清除相关缓存
	ctx := context.Background()
	if err := s.invalidateUserCache(ctx, id); err != nil {
		fmt.Printf("Warning: failed to invalidate cache after user update: %v\n", err)
	}

	return user, nil
}

func (s *CachedUserService) UpgradeMember(userID string, duration time.Duration) error {
	expireAt := time.Now().Add(duration)
	if err := s.repo.UpdateMemberStatus(userID, expireAt); err != nil {
		return err
	}

	// 清除相关缓存
	ctx := context.Background()
	if err := s.invalidateUserCache(ctx, userID); err != nil {
		fmt.Printf("Warning: failed to invalidate cache after member upgrade: %v\n", err)
	}

	return nil
}

// DeleteUser 删除用户（带缓存失效）
func (s *CachedUserService) DeleteUser(id string) error {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// 标记为已注销状态，而不是真正删除
	user.Status = model.StatusDeleted
	if err := s.repo.Update(user); err != nil {
		return err
	}

	// 清除相关缓存
	ctx := context.Background()
	if err := s.invalidateUserCache(ctx, id); err != nil {
		fmt.Printf("Warning: failed to invalidate cache after user deletion: %v\n", err)
	}

	return nil
}

// WarmupCache 预热缓存
func (s *CachedUserService) WarmupCache(ctx context.Context) error {
	// 预热热门用户数据
	popularUsers := []string{"1", "2", "3"} // 可以从配置或统计中获取

	for _, userID := range popularUsers {
		user, err := s.repo.GetByID(userID)
		if err != nil {
			continue // 跳过不存在的用户
		}

		cacheKey := s.getUserCacheKey(userID)
		if err := s.cache.Set(ctx, cacheKey, user, UserCacheTTL); err != nil {
			fmt.Printf("Warning: failed to warmup cache for user %s: %v\n", userID, err)
		}
	}

	// 预热第一页用户列表
	users, total, err := s.repo.GetList(0, 10)
	if err == nil {
		cacheKey := s.getUserListCacheKey(1, 10)
		result := struct {
			Users []model.User `json:"users"`
			Total int64        `json:"total"`
		}{
			Users: users,
			Total: total,
		}
		s.cache.Set(ctx, cacheKey, result, UserListCacheTTL)
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
func (s *CachedUserService) GetCacheStats(ctx context.Context) map[string]interface{} {
	stats := make(map[string]interface{})

	// 检查特定用户缓存
	testKeys := []string{"user:1", "user:2", "user_list:1:10"}
	existingKeys := 0

	for _, key := range testKeys {
		if exists, _ := s.cache.Exists(ctx, key); exists {
			existingKeys++
		}
	}

	stats["checked_keys"] = len(testKeys)
	stats["existing_keys"] = existingKeys
	stats["hit_rate"] = float64(existingKeys) / float64(len(testKeys))

	return stats
}
