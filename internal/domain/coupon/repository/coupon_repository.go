package repository

import (
	"errors"
	"user_crud_jwt/internal/domain/coupon/model"

	"gorm.io/gorm"
)

type CouponRepository interface {
	Create(coupon *model.Coupon) error
	GetByID(id uint) (*model.Coupon, error)
	DecreaseStock(couponID uint) error
	CreateUserCoupon(userCoupon *model.UserCoupon) error
	HasUserClaimed(userID, couponID uint) (bool, error)
}

type couponRepository struct {
	db *gorm.DB
}

func NewCouponRepository(db *gorm.DB) CouponRepository {
	return &couponRepository{db: db}
}

func (r *couponRepository) Create(coupon *model.Coupon) error {
	return r.db.Create(coupon).Error
}

func (r *couponRepository) GetByID(id uint) (*model.Coupon, error) {
	var coupon model.Coupon
	if err := r.db.First(&coupon, id).Error; err != nil {
		return nil, err
	}
	return &coupon, nil
}

// DecreaseStock 乐观锁扣减库存
func (r *couponRepository) DecreaseStock(couponID uint) error {
	result := r.db.Model(&model.Coupon{}).
		Where("id = ? AND stock > 0", couponID).
		UpdateColumn("stock", gorm.Expr("stock - 1"))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("insufficient stock")
	}
	return nil
}

func (r *couponRepository) CreateUserCoupon(userCoupon *model.UserCoupon) error {
	return r.db.Create(userCoupon).Error
}

func (r *couponRepository) HasUserClaimed(userID, couponID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.UserCoupon{}).
		Where("user_id = ? AND coupon_id = ?", userID, couponID).
		Count(&count).Error
	return count > 0, err
}
