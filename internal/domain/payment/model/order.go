package model

import (
	"encoding/json"
	"time"
	baseModel "user_crud_jwt/pkg/model"
)

// Order 订单模型
type Order struct {
	baseModel.BaseModel
	OrderNo     string          `json:"orderNo"`
	UserID      string          `json:"userId"`
	Amount      float64         `json:"amount"`
	Status      string          `json:"status"`  // pending, paid, cancelled, refunded
	Channel     string          `json:"channel"` // alipay, wechat
	Subject     string          `json:"subject"`
	ExtraParams json.RawMessage `json:"extraParams"`
	PaidAt      *time.Time      `json:"paidAt,omitempty"`
}

const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
	OrderStatusRefunded  = "refunded"

	ChannelAlipay = "alipay"
	ChannelWechat = "wechat"
)
