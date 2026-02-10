package model

import (
	"time"
	baseModel "user_crud_jwt/pkg/model"
)

// Coupon 优惠券定义
type Coupon struct {
	baseModel.BaseModel
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Total     int       `gorm:"not null" json:"total"`
	Stock     int       `gorm:"not null" json:"stock"` // 剩余库存
	Amount    float64   `gorm:"not null" json:"amount"`
	StartTime time.Time `gorm:"not null" json:"startTime"`
	EndTime   time.Time `gorm:"not null" json:"endTime"`
}

// UserCoupon 用户领取的优惠券
type UserCoupon struct {
	baseModel.BaseModel
	UserID   uint `gorm:"index;not null" json:"userId"`
	CouponID uint `gorm:"index;not null" json:"couponId"`
	Status   int  `gorm:"default:1" json:"status"` // 1:未使用, 2:已使用, 3:已过期
}
