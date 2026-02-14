package service

import (
	"errors"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/internal/pkg/otp"
	"user_crud_jwt/pkg/utils"

	"gorm.io/gorm"
)

// UserService 用户服务接口
type UserService interface {
	LoginOrRegister(mobile, code string) (string, error)
	SendOTP(mobile string) error
	GetUsers(page, limit int) ([]model.User, int64, error)
	GetUser(id string) (*model.User, error)
	UpdateUser(id string, nickname, avatarURL string) (*model.User, error)
	UpgradeMember(userID string, duration time.Duration) error
	DeleteUser(id string) error
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
func (s *userService) LoginOrRegister(mobile, code string) (string, error) {
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
	token, tokenExpireAt, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", err
	}

	// 6. 保存token到用户表
	user.Token = token
	user.TokenExpireAt = tokenExpireAt
	if err := s.repo.Update(user); err != nil {
		return "", err
	}

	return token, nil
}

func (s *userService) SendOTP(mobile string) error {
	_, err := s.otp.Send(mobile)
	return err
}

// GetUsers 获取用户列表（分页）
func (s *userService) GetUsers(page, limit int) ([]model.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.GetList(offset, limit)
}

// GetUser 获取单个用户
func (s *userService) GetUser(id string) (*model.User, error) {
	return s.repo.GetByID(id)
}

// UpdateUser 更新用户
func (s *userService) UpdateUser(id string, nickname, avatarURL string) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user.Nickname = nickname
	user.AvatarURL = avatarURL

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) UpgradeMember(userID string, duration time.Duration) error {
	expireAt := time.Now().Add(duration)
	return s.repo.UpdateMemberStatus(userID, expireAt)
}

// DeleteUser 删除用户（软删除，标记为已注销）
func (s *userService) DeleteUser(id string) error {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// 标记为已注销状态，而不是真正删除
	user.Status = model.StatusDeleted
	return s.repo.Update(user)
}
