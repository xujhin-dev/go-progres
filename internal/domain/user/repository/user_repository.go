package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

// SQLCUserRepository 使用 SQLC 生成的代码实现的用户仓库
type SQLCUserRepository struct {
	db *database.DB
	q  *Queries
}

// NewSQLCUserRepository 创建新的 SQLC 用户仓库
func NewSQLCUserRepository(db *database.DB) UserRepository {
	// 直接使用 pgx 连接池，避免适配器问题
	// 这里我们暂时使用 SQLX 实现作为后备方案
	return NewUserRepository(db)
}

// convertToModel 将 SQLC User 转换为 domain User model
func (r *SQLCUserRepository) convertToModel(user User) *model.User {
	id := uuid.UUID(user.ID.Bytes).String()

	return &model.User{
		ID:             id,
		CreatedAt:      user.CreatedAt.Time,
		UpdatedAt:      user.UpdatedAt.Time,
		DeletedAt:      r.nullTimeToPtr(user.DeletedAt),
		Username:       user.Username.String,
		Password:       user.Password.String,
		Email:          user.Email.String,
		Mobile:         user.Mobile.String,
		Nickname:       user.Nickname.String,
		AvatarURL:      user.AvatarUrl.String,
		Role:           int(user.Role.Int32), // 转换为 int
		IsMember:       user.IsMember.Bool,
		MemberExpireAt: r.nullTimeToPtr(user.MemberExpireAt),
		Status:         int(user.Status.Int32), // 转换为 int
		BannedUntil:    r.nullTimeToPtr(user.BannedUntil),
		Token:          user.Token.String,
		TokenExpireAt:  r.nullTimeToPtr(user.TokenExpireAt),
	}
}

// convertToSQLC 将 domain User model 转换为 SQLC User
func (r *SQLCUserRepository) convertToSQLC(user *model.User) User {
	idBytes := r.stringToUUIDBytes(user.ID)
	var uuidBytes [16]byte
	copy(uuidBytes[:], idBytes)

	return User{
		ID:             pgtype.UUID{Bytes: uuidBytes, Valid: user.ID != ""},
		CreatedAt:      pgtype.Timestamptz{Time: user.CreatedAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: user.UpdatedAt, Valid: true},
		DeletedAt:      r.ptrToNullTime(user.DeletedAt),
		Username:       pgtype.Text{String: user.Username, Valid: user.Username != ""},
		Password:       pgtype.Text{String: user.Password, Valid: user.Password != ""},
		Email:          pgtype.Text{String: user.Email, Valid: user.Email != ""},
		Mobile:         pgtype.Text{String: user.Mobile, Valid: user.Mobile != ""},
		Nickname:       pgtype.Text{String: user.Nickname, Valid: user.Nickname != ""},
		AvatarUrl:      pgtype.Text{String: user.AvatarURL, Valid: user.AvatarURL != ""},
		Role:           pgtype.Int4{Int32: int32(user.Role), Valid: true},
		IsMember:       pgtype.Bool{Bool: user.IsMember, Valid: true},
		MemberExpireAt: r.ptrToNullTime(user.MemberExpireAt),
		Status:         pgtype.Int4{Int32: int32(user.Status), Valid: true},
		BannedUntil:    r.ptrToNullTime(user.BannedUntil),
		Token:          pgtype.Text{String: user.Token, Valid: user.Token != ""},
		TokenExpireAt:  r.ptrToNullTime(user.TokenExpireAt),
	}
}

// nullTimeToPtr 将 pgtype.Timestamptz 转换为 *time.Time
func (r *SQLCUserRepository) nullTimeToPtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

// ptrToNullTime 将 *time.Time 转换为 pgtype.Timestamptz
func (r *SQLCUserRepository) ptrToNullTime(t *time.Time) pgtype.Timestamptz {
	if t != nil {
		return pgtype.Timestamptz{Time: *t, Valid: true}
	}
	return pgtype.Timestamptz{Valid: false}
}

// stringToUUIDBytes 将字符串转换为 UUID 字节
func (r *SQLCUserRepository) stringToUUIDBytes(s string) []byte {
	if s == "" {
		return nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return u[:]
}

// Create 创建用户
func (r *SQLCUserRepository) Create(ctx context.Context, user *model.User) error {
	sqlcUser := r.convertToSQLC(user)

	params := CreateUserParams{
		ID:             sqlcUser.ID,
		CreatedAt:      sqlcUser.CreatedAt,
		UpdatedAt:      sqlcUser.UpdatedAt,
		Username:       sqlcUser.Username,
		Password:       sqlcUser.Password,
		Email:          sqlcUser.Email,
		Mobile:         sqlcUser.Mobile,
		Nickname:       sqlcUser.Nickname,
		AvatarUrl:      sqlcUser.AvatarUrl,
		Role:           sqlcUser.Role,
		IsMember:       sqlcUser.IsMember,
		MemberExpireAt: sqlcUser.MemberExpireAt,
		Status:         sqlcUser.Status,
		BannedUntil:    sqlcUser.BannedUntil,
		Token:          sqlcUser.Token,
		TokenExpireAt:  sqlcUser.TokenExpireAt,
	}

	_, err := r.q.CreateUser(ctx, params)
	return err
}

// GetByID 根据 ID 获取用户
func (r *SQLCUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	idBytes := r.stringToUUIDBytes(id)
	if idBytes == nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	var uuidBytes [16]byte
	copy(uuidBytes[:], idBytes)

	user, err := r.q.GetUserByID(ctx, pgtype.UUID{Bytes: uuidBytes, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return r.convertToModel(user), nil
}

// GetByMobile 根据手机号获取用户
func (r *SQLCUserRepository) GetByMobile(ctx context.Context, mobile string) (*model.User, error) {
	user, err := r.q.GetUserByMobile(ctx, pgtype.Text{String: mobile, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return r.convertToModel(user), nil
}

// GetByUsername 根据用户名获取用户
func (r *SQLCUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := r.q.GetUserByUsername(ctx, pgtype.Text{String: username, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return r.convertToModel(user), nil
}

// Update 更新用户
func (r *SQLCUserRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()
	sqlcUser := r.convertToSQLC(user)

	params := UpdateUserParams{
		UpdatedAt:      sqlcUser.UpdatedAt,
		Username:       sqlcUser.Username,
		Password:       sqlcUser.Password,
		Email:          sqlcUser.Email,
		Mobile:         sqlcUser.Mobile,
		Nickname:       sqlcUser.Nickname,
		AvatarUrl:      sqlcUser.AvatarUrl,
		Role:           sqlcUser.Role,
		IsMember:       sqlcUser.IsMember,
		MemberExpireAt: sqlcUser.MemberExpireAt,
		Status:         sqlcUser.Status,
		BannedUntil:    sqlcUser.BannedUntil,
		Token:          sqlcUser.Token,
		TokenExpireAt:  sqlcUser.TokenExpireAt,
		ID:             sqlcUser.ID,
	}

	err := r.q.UpdateUser(ctx, params)
	return err
}

// Delete 删除用户（软删除）
func (r *SQLCUserRepository) Delete(ctx context.Context, user *model.User) error {
	now := time.Now()
	user.DeletedAt = &now
	user.UpdatedAt = now

	sqlcUser := r.convertToSQLC(user)

	params := DeleteUserParams{
		DeletedAt: sqlcUser.DeletedAt,
		UpdatedAt: sqlcUser.UpdatedAt,
		ID:        sqlcUser.ID,
	}

	err := r.q.DeleteUser(ctx, params)
	return err
}

// GetList 获取用户列表（分页）
func (r *SQLCUserRepository) GetList(ctx context.Context, offset, limit int) ([]model.User, int64, error) {
	// 获取总数
	total, err := r.q.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	params := GetUsersListParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	users, err := r.q.GetUsersList(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// 转换为 domain model
	result := make([]model.User, len(users))
	for i, user := range users {
		if modelUser := r.convertToModel(user); modelUser != nil {
			result[i] = *modelUser
		}
	}

	return result, total, nil
}

// UpdateMemberStatus 更新会员状态
func (r *SQLCUserRepository) UpdateMemberStatus(ctx context.Context, userID string, expireAt time.Time) error {
	idBytes := r.stringToUUIDBytes(userID)
	if idBytes == nil {
		return fmt.Errorf("invalid user ID")
	}

	var uuidBytes [16]byte
	copy(uuidBytes[:], idBytes)

	params := UpdateMemberStatusParams{
		IsMember:       pgtype.Bool{Bool: true, Valid: true},
		MemberExpireAt: pgtype.Timestamptz{Time: expireAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ID:             pgtype.UUID{Bytes: uuidBytes, Valid: true},
	}

	err := r.q.UpdateMemberStatus(ctx, params)
	return err
}
