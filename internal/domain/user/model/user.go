package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusNormal  = 0
	StatusBanned  = 1
	StatusDeleted = 2

	RoleUser  = 0
	RoleAdmin = 1
)

type User struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`

	Username       string     `json:"username"`
	Password       string     `json:"-"` // 密码不返回给前端
	Email          string     `json:"email"`
	Mobile         string     `gorm:"unique" json:"mobile"`
	Nickname       string     `json:"nickname"`
	AvatarURL      string     `json:"avatarUrl"`
	Role           int        `gorm:"default:0" json:"role"` // 0: 普通用户, 1: 管理员
	IsMember       bool       `gorm:"default:false" json:"isMember"`
	MemberExpireAt *time.Time `json:"memberExpireAt"`          // 会员过期时间
	Status         int        `gorm:"default:0" json:"status"` // 0:正常, 1:封禁, 2:注销
	BannedUntil    *time.Time `json:"bannedUntil"`             // 封禁截止时间
	Token          string     `gorm:"size:500" json:"-"`       // JWT Token，不返回给前端
	TokenExpireAt  *time.Time `json:"tokenExpireAt"`           // Token过期时间
}

// BeforeCreate 钩子：生成 UUID
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}
