package model

import (
	"time"
	baseModel "user_crud_jwt/pkg/model"
)

// Coupon 优惠券定义
type Coupon struct {
	baseModel.BaseModel
	Name      string    `json:"name"`
	Total     int       `json:"total"`
	Stock     int       `json:"stock"` // 剩余库存
	Amount    float64   `json:"amount"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// UserCoupon 用户领取的优惠券
type UserCoupon struct {
	baseModel.BaseModel
	UserID   string `json:"userId"`
	CouponID string `json:"couponId"`
	Status   int    `json:"status"` // 1:未使用, 2:已使用, 3:已过期
}
