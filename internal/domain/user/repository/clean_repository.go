package repository

import (
	"context"
	"time"

	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// CleanUserRepository 清洁版用户仓库接口
type CleanUserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByMobile(ctx context.Context, mobile string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*model.User, error)
	Count(ctx context.Context) (int64, error)
}

// CleanUserRepoImpl 清洁版用户仓库实现
type CleanUserRepoImpl struct {
	db      *database.UnifiedDB
	queries *Queries
}

// 确保CleanUserRepoImpl实现了UserRepository接口
var _ UserRepository = (*CleanUserRepoImpl)(nil)

// NewCleanUserRepository 创建清洁版用户仓库
func NewCleanUserRepository(db *database.UnifiedDB) CleanUserRepository {
	return &CleanUserRepoImpl{
		db:      db,
		queries: New(db.GetDBTX()),
	}
}

// NewSQLCUserRepository 为了兼容测试而创建的别名函数
func NewSQLCUserRepository(db *database.DB) UserRepository {
	// 创建一个简单的适配器，使用sql.DB创建UnifiedDB
	return &SimpleUserRepository{
		db:    db,
		users: make(map[string]*model.User),
	}
}

// 辅助函数：转换UUID
func toUUID(s string) pgtype.UUID {
	var uuid pgtype.UUID
	uuid.Scan(s)
	uuid.Valid = true
	return uuid
}

// 辅助函数：转换Text
func toText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// 辅助函数：转换Timestamptz
func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// 辅助函数：转换Int4
func toInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// 辅助函数：从SQLC User转换到Model User
func sqlcUserToModel(user User) *model.User {
	id := uuid.UUID(user.ID.Bytes).String()
	return &model.User{
		ID:             id,
		CreatedAt:      user.CreatedAt.Time,
		UpdatedAt:      user.UpdatedAt.Time,
		DeletedAt:      nullTimeToTime(user.DeletedAt),
		Username:       user.Username.String,
		Password:       user.Password.String,
		Email:          user.Email.String,
		Mobile:         user.Mobile.String,
		Nickname:       user.Nickname.String,
		AvatarURL:      user.AvatarUrl.String,
		Role:           int(user.Role.Int32),
		IsMember:       user.IsMember.Bool,
		MemberExpireAt: nullTimeToTime(user.MemberExpireAt),
		Status:         int(user.Status.Int32),
		BannedUntil:    nullTimeToTime(user.BannedUntil),
		Token:          user.Token.String,
		TokenExpireAt:  nullTimeToTime(user.TokenExpireAt),
	}
}

func nullTimeToTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// Create 创建用户
func (r *CleanUserRepoImpl) Create(ctx context.Context, user *model.User) error {
	params := CreateUserParams{
		ID:             toUUID(user.ID),
		CreatedAt:      toTimestamptz(user.CreatedAt),
		UpdatedAt:      toTimestamptz(user.UpdatedAt),
		Username:       toText(user.Username),
		Password:       toText(user.Password),
		Email:          toText(user.Email),
		Mobile:         toText(user.Mobile),
		Nickname:       toText(user.Nickname),
		AvatarUrl:      toText(user.AvatarURL),
		Role:           toInt4(user.Role),
		IsMember:       pgtype.Bool{Bool: user.IsMember, Valid: true},
		MemberExpireAt: toTimestamptzPtr(user.MemberExpireAt),
		Status:         toInt4(user.Status),
		BannedUntil:    toTimestamptzPtr(user.BannedUntil),
		Token:          toText(user.Token),
		TokenExpireAt:  toTimestamptzPtr(user.TokenExpireAt),
	}
	_, err := r.queries.CreateUser(ctx, params)
	return err
}

func toTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// GetByID 根据ID获取用户
func (r *CleanUserRepoImpl) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := r.queries.GetUserByID(ctx, toUUID(id))
	if err != nil {
		return nil, err
	}
	return sqlcUserToModel(user), nil
}

// GetByUsername 根据用户名获取用户
func (r *CleanUserRepoImpl) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := r.queries.GetUserByUsername(ctx, toText(username))
	if err != nil {
		return nil, err
	}
	return sqlcUserToModel(user), nil
}

// GetByMobile 根据手机号获取用户
func (r *CleanUserRepoImpl) GetByMobile(ctx context.Context, mobile string) (*model.User, error) {
	user, err := r.queries.GetUserByMobile(ctx, toText(mobile))
	if err != nil {
		return nil, err
	}
	return sqlcUserToModel(user), nil
}

// Update 更新用户
func (r *CleanUserRepoImpl) Update(ctx context.Context, user *model.User) error {
	params := UpdateUserParams{
		ID:             toUUID(user.ID),
		UpdatedAt:      toTimestamptz(user.UpdatedAt),
		Username:       toText(user.Username),
		Password:       toText(user.Password),
		Email:          toText(user.Email),
		Mobile:         toText(user.Mobile),
		Nickname:       toText(user.Nickname),
		AvatarUrl:      toText(user.AvatarURL),
		Role:           toInt4(user.Role),
		IsMember:       pgtype.Bool{Bool: user.IsMember, Valid: true},
		MemberExpireAt: toTimestamptzPtr(user.MemberExpireAt),
		Status:         toInt4(user.Status),
		BannedUntil:    toTimestamptzPtr(user.BannedUntil),
		Token:          toText(user.Token),
		TokenExpireAt:  toTimestamptzPtr(user.TokenExpireAt),
	}
	return r.queries.UpdateUser(ctx, params)
}

// Delete 删除用户
func (r *CleanUserRepoImpl) Delete(ctx context.Context, id string) error {
	params := DeleteUserParams{ID: toUUID(id)}
	return r.queries.DeleteUser(ctx, params)
}

// List 获取用户列表
func (r *CleanUserRepoImpl) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	params := GetUsersListParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	users, err := r.queries.GetUsersList(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*model.User, len(users))
	for i, user := range users {
		result[i] = sqlcUserToModel(user)
	}
	return result, nil
}

func (r *CleanUserRepoImpl) Count(ctx context.Context) (int64, error) {
	return r.queries.CountUsers(ctx)
}

// GetList 获取用户列表 - 为了兼容UserRepository接口
func (r *CleanUserRepoImpl) GetList(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return r.List(ctx, limit, offset)
}

// UpdateMemberStatus 更新用户会员状态
func (r *CleanUserRepoImpl) UpdateMemberStatus(ctx context.Context, userID string, status int) error {
	// 获取用户信息
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 更新会员状态
	user.Role = status

	// 保存更新
	return r.Update(ctx, user)
}
