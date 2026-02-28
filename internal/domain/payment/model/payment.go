package model

import (
	"time"
	baseModel "user_crud_jwt/pkg/model"
)

// Payment 支付记录模型
type Payment struct {
	baseModel.BaseModel
	OrderID      string    `json:"orderId"`
	Amount       float64   `json:"amount"`
	Method       string    `json:"method"`       // alipay, wechat, etc.
	TransactionID string   `json:"transactionId"` // 第三方支付交易号
	Status       string    `json:"status"`      // pending, success, failed
	PaidAt       *time.Time `json:"paidAt,omitempty"`
}

// Refund 退款记录模型
type Refund struct {
	baseModel.BaseModel
	OrderID       string    `json:"orderId"`
	PaymentID     string    `json:"paymentId"`
	Amount        float64   `json:"amount"`
	Reason        string    `json:"reason"`
	Status        string    `json:"status"`      // pending, success, failed
	RefundID      string    `json:"refundId"`    // 第三方退款单号
	ProcessedAt   *time.Time `json:"processedAt,omitempty"`
}
