package model

import (
	"time"
	baseModel "user_crud_jwt/pkg/model"
)

// User 用户模型
type User struct {
	baseModel.BaseModel
	Username       string     `gorm:"unique" json:"username"`
	Password       string     `json:"-"` // 密码不返回给前端
	Email          string     `gorm:"unique" json:"email"`
	Role           int        `gorm:"default:0" json:"role"` // 0: 普通用户, 1: 管理员
	IsMember       bool       `gorm:"default:false" json:"isMember"`
	MemberExpireAt *time.Time `json:"memberExpireAt,omitempty"`
	Status         int        `gorm:"default:0" json:"status"` // 0: 正常, 1: 封禁, 2: 注销
	BannedUntil    *time.Time `json:"bannedUntil,omitempty"`
}

const (
	RoleUser  = 0
	RoleAdmin = 1

	StatusNormal   = 0
	StatusBanned   = 1
	StatusDeleted  = 2
)
