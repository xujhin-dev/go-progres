package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型，替代 gorm.Model，使用驼峰命名的 JSON 标签
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}
