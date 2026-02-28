package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/payment/model"
	"user_crud_jwt/pkg/database"
)

// PaymentXRepository 使用 SQLX 实现的支付仓库
type PaymentXRepository struct {
	db *database.DB
}

// NewPaymentRepository 创建新的支付仓库
func NewPaymentRepository(db *database.DB) PaymentRepository {
	return &PaymentXRepository{db: db}
}

// CreateOrder 创建订单
func (r *PaymentXRepository) CreateOrder(order *model.Order) error {
	query := `
		INSERT INTO orders (
			id, created_at, updated_at, order_no, user_id, amount, status, channel, subject, extra_params, paid_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		order.ID, order.CreatedAt, order.UpdatedAt, order.OrderNo,
		order.UserID, order.Amount, order.Status, order.Channel,
		order.Subject, order.ExtraParams, order.PaidAt,
	)

	return err
}

// GetOrderByNo 根据订单号获取订单
func (r *PaymentXRepository) GetOrderByNo(orderNo string) (*model.Order, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, order_no, user_id, amount, status, 
			   channel, subject, extra_params, paid_at
		FROM orders 
		WHERE order_no = $1 AND deleted_at IS NULL`

	var order model.Order
	err := r.db.GetContext(context.Background(), &order, query, orderNo)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, err
	}

	return &order, nil
}

// UpdateOrderStatus 更新订单状态
func (r *PaymentXRepository) UpdateOrderStatus(orderNo string, status string, paidAt *time.Time, extra json.RawMessage) error {
	query := `
		UPDATE orders 
		SET status = $1, paid_at = $2, updated_at = $3, extra_params = $4
		WHERE order_no = $5 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(context.Background(), query, status, paidAt, time.Now(), extra, orderNo)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}
