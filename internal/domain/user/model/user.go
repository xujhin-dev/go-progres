package model

import "gorm.io/gorm"

// User 用户模型
type User struct {
	gorm.Model
	Username string `gorm:"unique" json:"username"`
	Password string `json:"-"` // 密码不返回给前端
	Email    string `gorm:"unique" json:"email"`
}
