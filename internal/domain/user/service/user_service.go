package service

import (
	"context"
	"errors"
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
	// 1. 验证验证码
	if !s.otp.Verify(mobile, code) {
		return "", errors.New("invalid verification code")
	}

	// 2. 查询用户是否存在
	user, err := s.repo.GetByMobile(ctx, mobile)
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

	return token, nil
}

// SendOTP 发送验证码
func (s *userService) SendOTP(ctx context.Context, mobile string) error {
	_, err := s.otp.Send(mobile)
	return err
}

// GetUsers 获取用户列表
func (s *userService) GetUsers(ctx context.Context, page, limit int) ([]model.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit
	return s.repo.GetList(ctx, offset, limit)
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

	return s.repo.UpdateMemberStatus(ctx, userID, expireAt)
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
