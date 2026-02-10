package service

import (
	"errors"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/pkg/utils"

	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务接口
type UserService interface {
	Register(username, password, email string) error
	Login(username, password string) (string, error)
	GetUsers(page, limit int) ([]model.User, int64, error)
	GetUser(id string) (*model.User, error)
	UpdateUser(id string, username, email string) (*model.User, error)
	ChangePassword(userID uint, oldPassword, newPassword string) error
	UpgradeMember(userID uint, duration time.Duration) error
	DeleteUser(id string) error
}

// userService 实现
type userService struct {
	repo repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

// Register 用户注册
func (s *userService) Register(username, password, email string) error {
	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     model.RoleUser, // 默认为普通用户
	}

	return s.repo.Create(user)
}

// Login 用户登录
func (s *userService) Login(username, password string) (string, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return "", errors.New("invalid username or password")
	}

	// 检查用户状态
	if user.Status == model.StatusBanned {
		if user.BannedUntil != nil && time.Now().After(*user.BannedUntil) {
			// 封禁已过期，自动解封
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

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid username or password")
	}

	// 生成Token (包含角色)
	return utils.GenerateToken(user.ID, user.Role)
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
func (s *userService) UpdateUser(id string, username, email string) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user.Username = username
	user.Email = email

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) UpgradeMember(userID uint, duration time.Duration) error {
	expireAt := time.Now().Add(duration)
	return s.repo.UpdateMemberStatus(userID, expireAt)
}

// ChangePassword 修改密码
func (s *userService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(string(userID))
	if err != nil {
		return err
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return s.repo.Update(user)
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
