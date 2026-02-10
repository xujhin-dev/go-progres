package repository

import (
	"encoding/json"
	"time"
	"user_crud_jwt/internal/domain/payment/model"

	"gorm.io/gorm"
)

type PaymentRepository interface {
	CreateOrder(order *model.Order) error
	GetOrderByNo(orderNo string) (*model.Order, error)
	UpdateOrderStatus(orderNo string, status string, paidAt *time.Time, extra json.RawMessage) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateOrder(order *model.Order) error {
	return r.db.Create(order).Error
}

func (r *paymentRepository) GetOrderByNo(orderNo string) (*model.Order, error) {
	var order model.Order
	if err := r.db.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *paymentRepository) UpdateOrderStatus(orderNo string, status string, paidAt *time.Time, extra json.RawMessage) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	if extra != nil {
		updates["extra_params"] = extra
	}
	return r.db.Model(&model.Order{}).Where("order_no = ?", orderNo).Updates(updates).Error
}
