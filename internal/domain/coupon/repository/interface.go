package repository

import (
	"context"
	"user_crud_jwt/internal/domain/coupon/model"
)

// CouponRepository 优惠券仓库接口
type CouponRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, coupon *model.Coupon) error
	CreateCoupon(ctx context.Context, coupon *model.Coupon) error
	GetByID(ctx context.Context, id string) (*model.Coupon, error)
	GetCouponByID(ctx context.Context, id string) (*model.Coupon, error)

	// 库存管理
	DecreaseStock(ctx context.Context, couponID string) error
	DecreaseCouponStock(ctx context.Context, couponID string) error

	// 用户优惠券管理
	CreateUserCoupon(ctx context.Context, userCoupon *model.UserCoupon) error
	GetUserCoupon(ctx context.Context, userID, couponID string) (*model.UserCoupon, error)
	HasUserClaimed(ctx context.Context, userID, couponID string) (bool, error)
	CountUserCoupons(ctx context.Context, userID, couponID string) (int64, error)
}
