package model

import (
	"encoding/json"
	baseModel "user_crud_jwt/pkg/model"
)

// Post 动态模型
type Post struct {
	baseModel.BaseModel
	UserID    uint            `json:"userId"`
	Content   string          `json:"content"`
	MediaURLs json.RawMessage `gorm:"type:jsonb" json:"mediaUrls"` // 存储图片/视频 URL 数组
	Type      string          `json:"type"`                        // text, image, video
	Status    string          `gorm:"default:'pending'" json:"status"` // pending, approved, rejected

	// 关联
	Comments []Comment `json:"comments,omitempty"`
	Topics   []Topic   `gorm:"many2many:post_topics;" json:"topics,omitempty"`
}

// Topic 话题模型
type Topic struct {
	baseModel.BaseModel
	Name string `gorm:"unique" json:"name"`
}

// Comment 评论模型
type Comment struct {
	baseModel.BaseModel
	PostID   uint   `json:"postId"`
	UserID   uint   `json:"userId"`
	Content  string `json:"content"`
	ParentID uint   `json:"parentId"` // 父评论ID (直接父评论)
	RootID   uint   `json:"rootId"`   // 根评论ID (一级评论ID，用于优化查询)
	Level    int    `gorm:"default:1" json:"level"` // 评论层级：1=一级评论，2=二级评论
}

// Like 点赞模型
type Like struct {
	baseModel.BaseModel
	UserID     uint   `json:"userId"`
	TargetID   uint   `json:"targetId"`
	TargetType string `json:"targetType"` // post, comment
}
