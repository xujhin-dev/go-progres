package repository

import (
	"context"
	"user_crud_jwt/internal/domain/user/model"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByMobile(ctx context.Context, mobile string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*model.User, error)
	GetList(ctx context.Context, limit, offset int) ([]*model.User, error)
	Count(ctx context.Context) (int64, error)
	UpdateMemberStatus(ctx context.Context, id string, status int) error
}
