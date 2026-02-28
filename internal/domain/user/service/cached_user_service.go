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

// LoginOrRegister 登录或注册（带缓存）
func (s *CachedUserService) LoginOrRegister(ctx context.Context, mobile, code string) (string, error) {
	// 1. 验证验证码
	if !s.otp.Verify(mobile, code) {
		return "", errors.New("invalid verification code")
	}

	// 2. 查询用户是否存在（先查缓存）
	cacheKey := fmt.Sprintf("user:mobile:%s", mobile)
	user, err := s.getUserFromCache(ctx, cacheKey)
	if err != nil {
		// 缓存未命中，从数据库查询
		user, err = s.repo.GetByMobile(ctx, mobile)
		if err != nil {
			if err.Error() == "user not found" {
				// 3. 不存在则注册
				user = model.NewUser(mobile, "User_"+mobile[len(mobile)-4:])

				if err := s.repo.Create(ctx, user); err != nil {
					return "", err
				}
			} else {
				return "", err
			}
		}

		// 缓存用户信息
		s.setUserCache(ctx, cacheKey, user, 5*time.Minute)
	}

	// 4. 检查用户状态
	if user.Status == model.StatusBanned {
		if user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
			return "", errors.New("account is banned")
		}
		// 封禁时间已过，解除封禁
		user.Status = model.StatusNormal
		user.BannedUntil = nil
	}

	if user.Status == model.StatusDeleted {
		return "", errors.New("account has been deleted")
	}

	// 5. 生成JWT token
	token, tokenExpireAt, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", err
	}

	// 6. 保存token到用户记录
	user.Token = token
	user.TokenExpireAt = tokenExpireAt
	if err := s.repo.Update(ctx, user); err != nil {
		return "", err
	}

	// 7. 更新缓存
	s.setUserCache(ctx, cacheKey, user, 5*time.Minute)
	s.setUserCache(ctx, fmt.Sprintf("user:id:%s", user.ID), user, 10*time.Minute)

	return token, nil
}

// SendOTP 发送验证码
func (s *CachedUserService) SendOTP(ctx context.Context, mobile string) error {
	_, err := s.otp.Send(mobile)
	return err
}

// GetUsers 获取用户列表（带缓存）
func (s *CachedUserService) GetUsers(ctx context.Context, page, limit int) ([]model.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	cacheKey := fmt.Sprintf("users:list:page:%d:limit:%d", page, limit)

	// 尝试从缓存获取
	var cachedUsers []model.User
	if err := s.cache.Get(ctx, cacheKey, &cachedUsers); err == nil {
		// 从缓存获取总数
		var total int64
		totalKey := fmt.Sprintf("users:total")
		if err := s.cache.Get(ctx, totalKey, &total); err == nil {
			return cachedUsers, total, nil
		}
	}

	offset := (page - 1) * limit
	users, err := s.repo.GetList(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 缓存结果
	s.cache.Set(ctx, cacheKey, users, 2*time.Minute)
	s.cache.Set(ctx, "users:total", total, 5*time.Minute)

	// 转换指针切片为值切片
	result := make([]model.User, len(users))
	for i, user := range users {
		result[i] = *user
	}

	return result, total, nil
}

// GetUser 获取用户信息（带缓存）
func (s *CachedUserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	cacheKey := fmt.Sprintf("user:id:%s", id)

	// 先从缓存获取
	var user model.User
	if err := s.cache.Get(ctx, cacheKey, &user); err == nil {
		return &user, nil
	}

	// 缓存未命中，从数据库查询
	dbUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 缓存用户信息
	s.setUserCache(ctx, cacheKey, dbUser, 10*time.Minute)

	return dbUser, nil
}

// UpdateUser 更新用户信息（带缓存）
func (s *CachedUserService) UpdateUser(ctx context.Context, id string, nickname, avatarURL string) (*model.User, error) {
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	if nickname != "" {
		user.Nickname = nickname
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	// 清除相关缓存
	s.clearUserCache(ctx, id)
	s.clearUserCache(ctx, fmt.Sprintf("user:mobile:%s", user.Mobile))

	return user, nil
}

// UpgradeMember 升级会员（带缓存）
func (s *CachedUserService) UpgradeMember(ctx context.Context, userID string, duration time.Duration) error {
	err := s.repo.UpdateMemberStatus(ctx, userID, 1) // 1 表示会员状态
	if err != nil {
		return err
	}

	// 清除用户缓存
	s.clearUserCache(ctx, userID)

	return nil
}

// DeleteUser 删除用户（带缓存）
func (s *CachedUserService) DeleteUser(ctx context.Context, id string) error {
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return err
	}

	user.Status = model.StatusDeleted
	err = s.repo.Update(ctx, user)
	if err != nil {
		return err
	}

	// 清除所有相关缓存
	s.clearUserCache(ctx, id)
	s.clearUserCache(ctx, fmt.Sprintf("user:mobile:%s", user.Mobile))

	return nil
}

// getUserFromCache 从缓存获取用户信息
func (s *CachedUserService) getUserFromCache(ctx context.Context, key string) (*model.User, error) {
	var user model.User
	if err := s.cache.Get(ctx, key, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// setUserCache 设置用户缓存
func (s *CachedUserService) setUserCache(ctx context.Context, key string, user *model.User, ttl time.Duration) {
	s.cache.Set(ctx, key, user, ttl)
}

// clearUserCache 清除用户相关缓存
func (s *CachedUserService) clearUserCache(ctx context.Context, id string) {
	keys := []string{
		fmt.Sprintf("user:id:%s", id),
	}

	// 清除用户信息缓存
	for _, key := range keys {
		s.cache.Delete(ctx, key)
	}

	// 清除用户列表缓存
	s.cache.Delete(ctx, "users:total")
}
