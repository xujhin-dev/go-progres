package repository

import (
	"time"
	"user_crud_jwt/internal/domain/user/model"

	"gorm.io/gorm"
)

// UserRepository 接口定义
type UserRepository interface {
	Create(user *model.User) error
	GetByID(id string) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetList(offset, limit int) ([]model.User, int64, error)
	Update(user *model.User) error
	UpdateMemberStatus(userID uint, expireAt time.Time) error
	Delete(user *model.User) error
}

// userRepository 实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建新的仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(id string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetList 获取用户列表（分页）
func (r *userRepository) GetList(offset, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if err := r.db.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// Update 更新用户
func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) UpdateMemberStatus(userID uint, expireAt time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_member":        true,
		"member_expire_at": expireAt,
	}).Error
}

// Delete 删除用户
func (r *userRepository) Delete(user *model.User) error {
	return r.db.Delete(user).Error
}
