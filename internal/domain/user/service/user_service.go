package service

import (
	"errors"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/pkg/utils"

	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务接口
type UserService interface {
	Register(username, password, email string) error
	Login(username, password string) (string, error)
	GetUsers() ([]model.User, error)
	GetUser(id string) (*model.User, error)
	UpdateUser(id string, username, email string) (*model.User, error)
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
	}

	return s.repo.Create(user)
}

// Login 用户登录
func (s *userService) Login(username, password string) (string, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return "", errors.New("invalid username or password")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid username or password")
	}

	// 生成Token
	return utils.GenerateToken(user.ID)
}

// GetUsers 获取所有用户
func (s *userService) GetUsers() ([]model.User, error) {
	return s.repo.GetAll()
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

// DeleteUser 删除用户
func (s *userService) DeleteUser(id string) error {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(user)
}
