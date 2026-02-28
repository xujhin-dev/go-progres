package model

import (
	"time"

	"github.com/google/uuid"
)

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// NewBaseModel 创建新的基础模型
func NewBaseModel() BaseModel {
	now := time.Now()
	return BaseModel{
		ID:        uuid.New().String(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
