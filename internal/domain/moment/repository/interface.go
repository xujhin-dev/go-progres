package repository

import (
	"context"
	"user_crud_jwt/internal/domain/moment/model"
)

// MomentRepository 动态仓库接口
type MomentRepository interface {
	// 动态相关
	CreatePost(ctx context.Context, post *model.Post) error
	GetPostByID(ctx context.Context, id string) (*model.Post, error)
	GetPosts(ctx context.Context, limit, offset int) ([]*model.Post, error)
	GetPostsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Post, error)
	UpdatePost(ctx context.Context, post *model.Post) error
	UpdatePostStatus(ctx context.Context, id string, status string) error
	DeletePost(ctx context.Context, id string) error
	
	// 评论相关
	CreateComment(ctx context.Context, comment *model.Comment) error
	GetCommentByID(ctx context.Context, id string) (*model.Comment, error)
	GetCommentsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.Comment, error)
	UpdateComment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, id string) error
	
	// 点赞相关
	CreateLike(ctx context.Context, like *model.Like) error
	DeleteLike(ctx context.Context, userID, targetID string, targetType string) error
	GetLikesByTarget(ctx context.Context, targetID string, targetType string, limit, offset int) ([]*model.Like, error)
	HasLiked(ctx context.Context, userID, targetID string) (bool, error)
	
	// 话题相关
	CreateTopic(ctx context.Context, topic *model.Topic) error
	GetTopicByID(ctx context.Context, id string) (*model.Topic, error)
	GetTopicByName(ctx context.Context, name string) (*model.Topic, error)
	GetTopicsByName(ctx context.Context, name string) ([]*model.Topic, error)
	GetTopics(ctx context.Context, limit, offset int) ([]*model.Topic, error)
	DeleteTopic(ctx context.Context, id string) error
	AssociatePostWithTopic(ctx context.Context, postID, topicID string) error
}
