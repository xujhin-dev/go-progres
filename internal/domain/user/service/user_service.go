package service

import (
	"context"
	"errors"
	"log"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/internal/pkg/otp"
	"user_crud_jwt/pkg/utils"
)

// UserService 用户服务接口
type UserService interface {
	LoginOrRegister(ctx context.Context, mobile, code string) (string, error)
	SendOTP(ctx context.Context, mobile string) error
	GetUsers(ctx context.Context, page, limit int) ([]model.User, int64, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, id string, nickname, avatarURL string) (*model.User, error)
	UpgradeMember(ctx context.Context, userID string, duration time.Duration) error
	DeleteUser(ctx context.Context, id string) error
}

// userService 实现
type userService struct {
	repo repository.UserRepository
	otp  otp.OTPService
}

// NewUserService 创建用户服务
func NewUserService(repo repository.UserRepository, otp otp.OTPService) UserService {
	return &userService{repo: repo, otp: otp}
}

// LoginOrRegister 登录或注册
func (s *userService) LoginOrRegister(ctx context.Context, mobile, code string) (string, error) {
	log.Printf("[UserService] LoginOrRegister called with mobile: %s, code: %s", mobile, code)

	// 1. 验证验证码
	log.Printf("[UserService] Verifying OTP code...")
	if !s.otp.Verify(mobile, code) {
		log.Printf("[UserService] OTP verification failed for mobile: %s", mobile)
		return "", errors.New("invalid verification code")
	}
	log.Printf("[UserService] OTP verification successful for mobile: %s", mobile)

	// 2. 查询用户是否存在
	log.Printf("[UserService] Checking if user exists for mobile: %s", mobile)
	user, err := s.repo.GetByMobile(ctx, mobile)
	if err != nil {
		log.Printf("[UserService] Database error when getting user: %v", err)
		return "", err
	}

	if user == nil {
		log.Printf("[UserService] User not found, creating new user for mobile: %s", mobile)
		// 3. 不存在则注册
		user = model.NewUser(mobile, "User_"+mobile[len(mobile)-4:])
		log.Printf("[UserService] Created new user with ID: %s", user.ID)

		if err := s.repo.Create(ctx, user); err != nil {
			log.Printf("[UserService] Failed to create user: %v", err)
			return "", err
		}
		log.Printf("[UserService] User created successfully")
	} else {
		log.Printf("[UserService] Found existing user with ID: %s", user.ID)
	}

	// 4. 检查用户状态
	log.Printf("[UserService] Checking user status - Status: %d, Role: %d", user.Status, user.Role)
	if user.Status == model.StatusBanned {
		if user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
			log.Printf("[UserService] Account is banned until: %v", user.BannedUntil)
			return "", errors.New("account is banned")
		}
		// 封禁时间已过，解除封禁
		log.Printf("[UserService] Ban expired, lifting ban")
		user.Status = model.StatusNormal
		user.BannedUntil = nil
	}

	if user.Status == model.StatusDeleted {
		log.Printf("[UserService] Account has been deleted")
		return "", errors.New("account has been deleted")
	}

	// 5. 生成JWT token
	log.Printf("[UserService] Generating JWT token for user ID: %s", user.ID)
	token, tokenExpireAt, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		log.Printf("[UserService] Failed to generate JWT token: %v", err)
		return "", err
	}
	log.Printf("[UserService] JWT token generated successfully")

	// 6. 保存token到用户记录
	user.Token = token
	user.TokenExpireAt = tokenExpireAt
	log.Printf("[UserService] Updating user record with token")
	if err := s.repo.Update(ctx, user); err != nil {
		log.Printf("[UserService] Failed to update user record: %v", err)
		return "", err
	}
	log.Printf("[UserService] User record updated successfully")

	log.Printf("[UserService] Login process completed successfully")
	return token, nil
}

// SendOTP 发送验证码
func (s *userService) SendOTP(ctx context.Context, mobile string) error {
	_, err := s.otp.Send(mobile)
	return err
}

// GetUsers 获取用户列表
func (s *userService) GetUsers(ctx context.Context, page, limit int) ([]model.User, int64, error) {
	// 获取总数
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	users, err := s.repo.GetList(ctx, limit, offset) // 修正参数顺序
	if err != nil {
		return nil, 0, err
	}

	// 转换指针切片为值切片
	result := make([]model.User, len(users))
	for i, user := range users {
		result[i] = *user
	}

	return result, total, nil
}

// GetUser 获取用户信息
func (s *userService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateUser 更新用户信息
func (s *userService) UpdateUser(ctx context.Context, id string, nickname, avatarURL string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
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

	return user, nil
}

// UpgradeMember 升级会员
func (s *userService) UpgradeMember(ctx context.Context, userID string, duration time.Duration) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	var expireAt time.Time
	if user.MemberExpireAt != nil && user.MemberExpireAt.After(time.Now()) {
		// 如果已经是会员，在原有时间基础上延长
		expireAt = user.MemberExpireAt.Add(duration)
	} else {
		// 如果不是会员，从现在开始计算
		expireAt = time.Now().Add(duration)
	}

	// 更新用户的会员过期时间
	user.MemberExpireAt = &expireAt
	user.IsMember = true
	user.Status = model.StatusNormal

	return s.repo.Update(ctx, user)
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(ctx context.Context, id string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	user.Status = model.StatusDeleted
	return s.repo.Update(ctx, user)
}
