package repository

import (
	"context"
	"user_crud_jwt/internal/domain/payment/model"
)

// PaymentRepository 支付仓库接口
type PaymentRepository interface {
	// 订单相关
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrderByID(ctx context.Context, id string) (*model.Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*model.Order, error)
	GetOrdersByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Order, error)
	UpdateOrder(ctx context.Context, order *model.Order) error
	UpdateOrderStatus(ctx context.Context, id string, status string) error
	
	// 支付相关
	CreatePayment(ctx context.Context, payment *model.Payment) error
	GetPaymentByOrderID(ctx context.Context, orderID string) (*model.Payment, error)
	UpdatePaymentStatus(ctx context.Context, id string, status string) error
	
	// 退款相关
	CreateRefund(ctx context.Context, refund *model.Refund) error
	GetRefundByOrderID(ctx context.Context, orderID string) (*model.Refund, error)
	UpdateRefundStatus(ctx context.Context, id string, status string) error
}
