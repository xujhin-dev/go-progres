package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	couponModel "user_crud_jwt/internal/domain/coupon/model"
	"user_crud_jwt/pkg/database"
	"user_crud_jwt/pkg/model"

	"github.com/jackc/pgx/v5/pgtype"
)

// SQLCCouponRepository 使用 sqlc 生成的代码实现的优惠券仓库
type SQLCCouponRepository struct {
	db      *database.DB
	queries *Queries
}

// NewSQLCCouponRepository 创建新的基于 sqlc 的优惠券仓库
func NewSQLCCouponRepository(db *database.DB) *SQLCCouponRepository {
	// TODO: 修复 SQLC 兼容性问题
	// 使用 DB 的底层连接，SQLC生成的代码需要的是符合 DBTX 接口的对象
	return &SQLCCouponRepository{
		db:      db,
		queries: nil, // 暂时设为 nil，后续修复
	}
}

// 辅助转换函数
func stringToText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func int32ToPgtype(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

func nullTimeToTime(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func timeToTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func timeToNullTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func uuidToPgtype(s string) pgtype.UUID {
	var uuid pgtype.UUID
	if s != "" {
		uuid.Scan(s)
		uuid.Valid = true
	}
	return uuid
}

// Create 创建优惠券 (实现接口方法)
func (r *SQLCCouponRepository) Create(coupon *couponModel.Coupon) error {
	return r.CreateCoupon(context.Background(), coupon)
}

// CreateCoupon 创建优惠券
func (r *SQLCCouponRepository) CreateCoupon(ctx context.Context, coupon *couponModel.Coupon) error {
	// 将 float64 转换为 big.Int 用于 pgtype.Numeric
	amountInt := big.NewInt(int64(coupon.Amount * 100))
	params := CreateCouponParams{
		ID:        uuidToPgtype(coupon.ID),
		CreatedAt: timeToTimestamptz(coupon.CreatedAt),
		UpdatedAt: timeToTimestamptz(coupon.UpdatedAt),
		Name:      stringToText(coupon.Name),
		Total:     int32ToPgtype(coupon.Total),
		Stock:     int32ToPgtype(coupon.Stock),
		Amount:    pgtype.Numeric{Int: amountInt, Valid: true}, // 转换为分
		StartTime: timeToTimestamptz(coupon.StartTime),
		EndTime:   timeToTimestamptz(coupon.EndTime),
	}
	_, err := r.queries.CreateCoupon(ctx, params)
	return err
}

// GetByID 根据 ID 获取优惠券 (实现接口方法)
func (r *SQLCCouponRepository) GetByID(id string) (*couponModel.Coupon, error) {
	return r.GetCouponByID(context.Background(), id)
}

// GetCouponByID 根据 ID 获取优惠券
func (r *SQLCCouponRepository) GetCouponByID(ctx context.Context, id string) (*couponModel.Coupon, error) {
	uuid := uuidToPgtype(id)
	coupon, err := r.queries.GetCouponByID(ctx, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("coupon not found")
		}
		return nil, err
	}

	// 转换 sqlc Coupon 到 model Coupon
	modelCoupon := &couponModel.Coupon{
		BaseModel: model.BaseModel{
			ID:        id,
			CreatedAt: coupon.CreatedAt.Time,
			UpdatedAt: coupon.UpdatedAt.Time,
			DeletedAt: nullTimeToTime(coupon.DeletedAt),
		},
		Name:      coupon.Name,
		Total:     int(coupon.Total),
		Stock:     int(coupon.Stock),
		Amount:    float64(coupon.Amount.Int.Int64()) / 100.0, // 从分转换回元
		StartTime: coupon.StartTime.Time,
		EndTime:   coupon.EndTime.Time,
	}
	return modelCoupon, nil
}

// DecreaseStock 减少优惠券库存 (实现接口方法)
func (r *SQLCCouponRepository) DecreaseStock(couponID string) error {
	return r.DecreaseCouponStock(context.Background(), couponID)
}

// DecreaseCouponStock 减少优惠券库存
func (r *SQLCCouponRepository) DecreaseCouponStock(ctx context.Context, couponID string) error {
	uuid := uuidToPgtype(couponID)
	params := DecreaseCouponStockParams{
		UpdatedAt: timeToTimestamptz(time.Now()),
		ID:        uuid,
	}
	return r.queries.DecreaseCouponStock(ctx, params)
}

// GetUserCoupon 获取用户优惠券
func (r *SQLCCouponRepository) GetUserCoupon(ctx context.Context, userID, couponID string) (*couponModel.UserCoupon, error) {
	userUUID := uuidToPgtype(userID)
	couponUUID := uuidToPgtype(couponID)
	params := GetUserCouponParams{
		UserID:   userUUID,
		CouponID: couponUUID,
	}
	userCoupon, err := r.queries.GetUserCoupon(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user coupon not found")
		}
		return nil, err
	}

	// 转换 sqlc UserCoupon 到 model UserCoupon
	modelUserCoupon := &couponModel.UserCoupon{
		BaseModel: model.BaseModel{
			ID:        userID, // 临时设置，需要从 sqlc 结果获取
			CreatedAt: userCoupon.CreatedAt.Time,
			UpdatedAt: userCoupon.UpdatedAt.Time,
			DeletedAt: nullTimeToTime(userCoupon.DeletedAt),
		},
		UserID:   userID,
		CouponID: couponID,
		Status:   int(userCoupon.Status.Int32),
	}
	return modelUserCoupon, nil
}

// CreateUserCoupon 创建用户优惠券
func (r *SQLCCouponRepository) CreateUserCoupon(ctx context.Context, userCoupon *couponModel.UserCoupon) error {
	params := CreateUserCouponParams{
		ID:        uuidToPgtype(userCoupon.ID),
		CreatedAt: timeToTimestamptz(userCoupon.CreatedAt),
		UpdatedAt: timeToTimestamptz(userCoupon.UpdatedAt),
		UserID:    uuidToPgtype(userCoupon.UserID),
		CouponID:  uuidToPgtype(userCoupon.CouponID),
		Status:    int32ToPgtype(userCoupon.Status),
	}
	_, err := r.queries.CreateUserCoupon(ctx, params)
	return err
}

// CountUserCoupons 统计用户优惠券数量
func (r *SQLCCouponRepository) CountUserCoupons(ctx context.Context, userID, couponID string) (int64, error) {
	userUUID := uuidToPgtype(userID)
	couponUUID := uuidToPgtype(couponID)
	params := CountUserCouponsParams{
		UserID:   userUUID,
		CouponID: couponUUID,
	}
	return r.queries.CountUserCoupons(ctx, params)
}

// HasUserClaimed 检查用户是否已领取优惠券 (实现接口方法)
func (r *SQLCCouponRepository) HasUserClaimed(userID, couponID string) (bool, error) {
	count, err := r.CountUserCoupons(context.Background(), userID, couponID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
