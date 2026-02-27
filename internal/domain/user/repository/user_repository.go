package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"
)

// UserRepository 接口定义
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByMobile(ctx context.Context, mobile string) (*model.User, error)
	GetList(ctx context.Context, offset, limit int) ([]model.User, int64, error)
	Update(ctx context.Context, user *model.User) error
	UpdateMemberStatus(ctx context.Context, userID string, expireAt time.Time) error
	Delete(ctx context.Context, user *model.User) error
}

// userRepository 实现
type userRepository struct {
	db *database.DB
}

// NewUserRepository 创建新的仓库实例
func NewUserRepository(db *database.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (
			id, created_at, updated_at, username, password, email, mobile, 
			nickname, avatar_url, role, is_member, member_expire_at, 
			status, banned_until, token, token_expire_at
		) VALUES (
			:id, :created_at, :updated_at, :username, :password, :email, :mobile,
			:nickname, :avatar_url, :role, :is_member, :member_expire_at,
			:status, :banned_until, :token, :token_expire_at
		)`

	_, err := r.db.NamedExec(ctx, query, user)
	return err
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile,
			   nickname, avatar_url, role, is_member, member_expire_at, status, 
			   banned_until, token, token_expire_at
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`

	var user model.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile,
			   nickname, avatar_url, role, is_member, member_expire_at, status, 
			   banned_until, token, token_expire_at
		FROM users 
		WHERE username = $1 AND deleted_at IS NULL`

	var user model.User
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByMobile 根据手机号获取用户
func (r *userRepository) GetByMobile(ctx context.Context, mobile string) (*model.User, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile,
			   nickname, avatar_url, role, is_member, member_expire_at, status, 
			   banned_until, token, token_expire_at
		FROM users 
		WHERE mobile = $1 AND deleted_at IS NULL`

	var user model.User
	err := r.db.GetContext(ctx, &user, query, mobile)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetList 获取用户列表（分页）
func (r *userRepository) GetList(ctx context.Context, offset, limit int) ([]model.User, int64, error) {
	// 获取总数
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	query := `
		SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile,
			   nickname, avatar_url, role, is_member, member_expire_at, status, 
			   banned_until, token, token_expire_at
		FROM users 
		WHERE deleted_at IS NULL 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`

	var users []model.User
	err = r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Update 更新用户
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET 
			updated_at = :updated_at, username = :username, password = :password, 
			email = :email, mobile = :mobile, nickname = :nickname, 
			avatar_url = :avatar_url, role = :role, is_member = :is_member, 
			member_expire_at = :member_expire_at, status = :status, 
			banned_until = :banned_until, token = :token, token_expire_at = :token_expire_at
		WHERE id = :id AND deleted_at IS NULL`

	result, err := r.db.NamedExec(ctx, query, user)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}

// UpdateMemberStatus 更新会员状态
func (r *userRepository) UpdateMemberStatus(ctx context.Context, userID string, expireAt time.Time) error {
	query := `
		UPDATE users 
		SET is_member = true, member_expire_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, expireAt, time.Now(), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete 删除用户（软删除）
func (r *userRepository) Delete(ctx context.Context, user *model.User) error {
	now := time.Now()
	user.DeletedAt = &now
	user.UpdatedAt = now

	query := `UPDATE users SET deleted_at = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query, now, now, user.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
