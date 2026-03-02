package model

import (
	"encoding/json"
	baseModel "user_crud_jwt/pkg/model"
)

// UserInfo 用户信息摘要
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Mobile   string `json:"mobile"`
	Avatar   string `json:"avatar"`
	Role     int    `json:"role"`
}

// Post 动态模型
type Post struct {
	baseModel.BaseModel
	UserID    string          `json:"userId"`
	Content   string          `json:"content"`
	MediaURLs json.RawMessage `json:"mediaUrls"` // 存储图片/视频 URL 数组
	Type      string          `json:"type"`      // text, image, video
	Status    string          `json:"status"`    // pending, approved, rejected

	// 关联
	Comments []Comment `json:"comments,omitempty"`
	Topics   []Topic   `json:"topics,omitempty"`
}

// Topic 话题模型
type Topic struct {
	baseModel.BaseModel
	Name string `json:"name"`
}

// Comment 评论模型
type Comment struct {
	baseModel.BaseModel
	PostID   string `json:"postId"`
	UserID   string `json:"userId"`
	Content  string `json:"content"`
	ParentID string `json:"parentId"` // 父评论ID (直接父评论)
	RootID   string `json:"rootId"`   // 根评论ID (一级评论ID，用于优化查询)
	Level    int    `json:"level"`    // 评论层级：1=一级评论，2=二级评论

	// 关联用户信息
	User *UserInfo `json:"user,omitempty"`

	// 回复统计
	ReplyCount int `json:"replyCount"`
}

// Like 点赞模型
type Like struct {
	baseModel.BaseModel
	UserID     string `json:"userId"`
	TargetID   string `json:"targetId"`
	TargetType string `json:"targetType"` // post, comment

	// 关联用户信息
	User *UserInfo `json:"user,omitempty"`
}

// LikeWithUser 带用户信息的点赞响应
type LikeWithUser struct {
	Like
	IsLiked bool `json:"isLiked"` // 当前用户是否已点赞
}

// CommentWithUser 带用户信息的评论响应
type CommentWithUser struct {
	Comment
	Replies []CommentWithUser `json:"replies,omitempty"` // 子评论
}

// PostWithDetail 带详细信息的动态响应
type PostWithDetail struct {
	Post
	User      *UserInfo         `json:"user,omitempty"`     // 发布用户信息
	Comments  []CommentWithUser `json:"comments,omitempty"` // 评论列表
	Topics    []Topic           `json:"topics,omitempty"`   // 话题列表
	LikeCount int               `json:"likeCount"`          // 点赞数
	IsLiked   bool              `json:"isLiked"`            // 当前用户是否已点赞
}
