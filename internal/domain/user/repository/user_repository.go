package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"
)

// UserRepository 使用 SQLX 实现的用户仓库
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

// UserXRepository 使用 SQLX 实现的用户仓库
type UserXRepository struct {
	db *database.DB
}

// NewUserRepository 创建新的用户仓库
func NewUserRepository(db *database.DB) UserRepository {
	return &UserXRepository{db: db}
}

// Create 创建用户
func (r *UserXRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (
			id, created_at, updated_at, username, password, email, mobile, 
			nickname, avatar_url, role, is_member, member_expire_at, status, 
			banned_until, token, token_expire_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.CreatedAt, user.UpdatedAt, user.Username, user.Password,
		user.Email, user.Mobile, user.Nickname, user.AvatarURL, user.Role,
		user.IsMember, user.MemberExpireAt, user.Status, user.BannedUntil,
		user.Token, user.TokenExpireAt,
	)

	return err
}

// GetByID 根据 ID 获取用户
func (r *UserXRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, username, password, email, mobile, 
			   nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, 
			   token, token_expire_at
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
func (r *UserXRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, username, password, email, mobile, 
			   nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, 
			   token, token_expire_at
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
func (r *UserXRepository) GetByMobile(ctx context.Context, mobile string) (*model.User, error) {
	query := `
		SELECT 
			id::text, 
			COALESCE(created_at, NOW()) as created_at, 
			COALESCE(updated_at, NOW()) as updated_at, 
			deleted_at, 
			COALESCE(username, '') as username, 
			COALESCE(password, '') as password, 
			COALESCE(email, '') as email, 
			mobile, 
			COALESCE(nickname, '') as nickname, 
			COALESCE(avatar_url, '') as avatar_url, 
			COALESCE(role, 0) as role, 
			COALESCE(is_member, false) as is_member, 
			member_expire_at, 
			COALESCE(status, 0) as status, 
			banned_until, 
			COALESCE(token, '') as token, 
			token_expire_at
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

// Update 更新用户
func (r *UserXRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET 
			updated_at = $1, username = $2, password = $3, email = $4, mobile = $5, 
			nickname = $6, avatar_url = $7, role = $8, is_member = $9, member_expire_at = $10, 
			status = $11, banned_until = $12, token = $13, token_expire_at = $14
		WHERE id = $15 AND deleted_at IS NULL`

	user.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		user.UpdatedAt, user.Username, user.Password, user.Email, user.Mobile,
		user.Nickname, user.AvatarURL, user.Role, user.IsMember, user.MemberExpireAt,
		user.Status, user.BannedUntil, user.Token, user.TokenExpireAt, user.ID,
	)

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

// Delete 删除用户（软删除）
func (r *UserXRepository) Delete(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users 
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	now := time.Now()
	user.DeletedAt = &now
	user.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, query, user.DeletedAt, user.UpdatedAt, user.ID)
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

// GetList 获取用户列表（分页）
func (r *UserXRepository) GetList(ctx context.Context, offset, limit int) ([]model.User, int64, error) {
	// 获取总数
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	listQuery := `
		SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, 
			   nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, 
			   token, token_expire_at
		FROM users 
		WHERE deleted_at IS NULL 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`

	var users []model.User
	err = r.db.SelectContext(ctx, &users, listQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateMemberStatus 更新会员状态
func (r *UserXRepository) UpdateMemberStatus(ctx context.Context, userID string, expireAt time.Time) error {
	query := `
		UPDATE users 
		SET is_member = $1, member_expire_at = $2, updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, true, expireAt, time.Now(), userID)
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
