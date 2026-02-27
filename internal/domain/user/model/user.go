package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusNormal  = 0
	StatusBanned  = 1
	StatusDeleted = 2

	RoleUser  = 0
	RoleAdmin = 1
)

type User struct {
	ID        string     `db:"id" json:"id"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt *time.Time `db:"deleted_at" json:"deletedAt,omitempty"`

	Username       string     `db:"username" json:"username"`
	Password       string     `db:"password" json:"-"` // 密码不返回给前端
	Email          string     `db:"email" json:"email"`
	Mobile         string     `db:"mobile" json:"mobile"`
	Nickname       string     `db:"nickname" json:"nickname"`
	AvatarURL      string     `db:"avatar_url" json:"avatarUrl"`
	Role           int        `db:"role" json:"role"` // 0: 普通用户, 1: 管理员
	IsMember       bool       `db:"is_member" json:"isMember"`
	MemberExpireAt *time.Time `db:"member_expire_at" json:"memberExpireAt"` // 会员过期时间
	Status         int        `db:"status" json:"status"`                   // 0:正常, 1:封禁, 2:注销
	BannedUntil    *time.Time `db:"banned_until" json:"bannedUntil"`        // 封禁截止时间
	Token          string     `db:"token" json:"-"`                         // JWT Token，不返回给前端
	TokenExpireAt  *time.Time `db:"token_expire_at" json:"tokenExpireAt"`   // Token过期时间
}

// NewUser 创建新用户实例
func NewUser(mobile, nickname string) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New().String(),
		CreatedAt: now,
		UpdatedAt: now,
		Mobile:    mobile,
		Nickname:  nickname,
		Role:      RoleUser,
		Status:    StatusNormal,
		IsMember:  false,
	}
}

// TableName 返回表名
func (u *User) TableName() string {
	return "users"
}
