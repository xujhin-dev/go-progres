package repository

import (
	"context"
	"sync"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"

	"github.com/google/uuid"
)

// SimpleUserRepository 简单的用户仓库实现（使用内存存储）
type SimpleUserRepository struct {
	db    *database.DB
	users map[string]*model.User // 手机号 -> 用户
	mutex sync.RWMutex
}

// NewSimpleUserRepository 创建简单用户仓库
func NewSimpleUserRepository(db *database.DB) UserRepository {
	return &SimpleUserRepository{
		db:    db,
		users: make(map[string]*model.User),
	}
}

func (r *SimpleUserRepository) Create(ctx context.Context, user *model.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 确保用户有ID
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	r.users[user.Mobile] = user
	return nil
}

func (r *SimpleUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, nil
}

func (r *SimpleUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, user := range r.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

func (r *SimpleUserRepository) GetByMobile(ctx context.Context, mobile string) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[mobile]
	if !exists {
		return nil, nil // 用户不存在
	}
	return user, nil
}

func (r *SimpleUserRepository) Update(ctx context.Context, user *model.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if existingUser, exists := r.users[user.Mobile]; exists {
		// 更新现有用户，保持ID不变
		user.ID = existingUser.ID
		r.users[user.Mobile] = user
	}
	return nil
}

func (r *SimpleUserRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for mobile, user := range r.users {
		if user.ID == id {
			delete(r.users, mobile)
			break
		}
	}
	return nil
}

func (r *SimpleUserRepository) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var users []*model.User
	count := 0

	// 遍历所有用户
	for _, user := range r.users {
		if count >= offset {
			if len(users) < limit {
				users = append(users, user)
			} else {
				break // 已达到limit数量
			}
		}
		count++
	}
	return users, nil
}

func (r *SimpleUserRepository) GetList(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return r.List(ctx, limit, offset)
}

func (r *SimpleUserRepository) Count(ctx context.Context) (int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return int64(len(r.users)), nil
}

func (r *SimpleUserRepository) UpdateMemberStatus(ctx context.Context, id string, status int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, user := range r.users {
		if user.ID == id {
			user.Role = status
			break
		}
	}
	return nil
}
