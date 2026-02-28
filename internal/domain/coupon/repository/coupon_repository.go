package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/coupon/model"
	"user_crud_jwt/pkg/database"
)

// CouponXRepository 使用 SQLX 实现的优惠券仓库
type CouponXRepository struct {
	db *database.DB
}

// NewCouponRepository 创建新的优惠券仓库
func NewCouponRepository(db *database.DB) CouponRepository {
	return &CouponXRepository{db: db}
}

// Create 创建优惠券
func (r *CouponXRepository) Create(coupon *model.Coupon) error {
	query := `
		INSERT INTO coupons (
			id, created_at, updated_at, name, total, stock, amount, start_time, end_time
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		coupon.ID, coupon.CreatedAt, coupon.UpdatedAt, coupon.Name,
		coupon.Total, coupon.Stock, coupon.Amount, coupon.StartTime, coupon.EndTime,
	)

	return err
}

// GetByID 根据 ID 获取优惠券
func (r *CouponXRepository) GetByID(id string) (*model.Coupon, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, name, total, stock, amount, start_time, end_time
		FROM coupons 
		WHERE id = $1 AND deleted_at IS NULL`

	var coupon model.Coupon
	err := r.db.GetContext(context.Background(), &coupon, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("coupon not found")
		}
		return nil, err
	}

	return &coupon, nil
}

// DecreaseStock 减少优惠券库存
func (r *CouponXRepository) DecreaseStock(couponID string) error {
	query := `
		UPDATE coupons 
		SET stock = stock - 1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL AND stock > 0`

	result, err := r.db.ExecContext(context.Background(), query, time.Now(), couponID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("coupon not found or insufficient stock")
	}

	return nil
}

// CreateUserCoupon 创建用户优惠券关联
func (r *CouponXRepository) CreateUserCoupon(userCoupon *model.UserCoupon) error {
	query := `
		INSERT INTO user_coupons (
			id, created_at, updated_at, user_id, coupon_id, status
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		userCoupon.ID, userCoupon.CreatedAt, userCoupon.UpdatedAt,
		userCoupon.UserID, userCoupon.CouponID, userCoupon.Status,
	)

	return err
}

// HasUserClaimed 检查用户是否已领取优惠券
func (r *CouponXRepository) HasUserClaimed(userID, couponID string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM user_coupons 
		WHERE user_id = $1 AND coupon_id = $2 AND deleted_at IS NULL`

	var count int64
	err := r.db.GetContext(context.Background(), &count, query, userID, couponID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
