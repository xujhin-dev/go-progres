package repository

import (
	"user_crud_jwt/internal/domain/moment/model"

	"gorm.io/gorm"
)

type MomentRepository interface {
	CreatePost(post *model.Post) error
	GetPostByID(id string) (*model.Post, error)
	GetPosts(status string, offset, limit int) ([]model.Post, int64, error)
	UpdatePostStatus(id string, status string) error

	CreateComment(comment *model.Comment) error
	GetCommentByID(id string) (*model.Comment, error)
	GetCommentsByPostID(postID string, offset, limit int) ([]model.Comment, int64, error)

	CreateLike(like *model.Like) error
	DeleteLike(userID, targetID string, targetType string) error
	HasLiked(userID, targetID string, targetType string) (bool, error)

	GetTopicByName(name string) (*model.Topic, error)
	CreateTopic(topic *model.Topic) error
	GetTopics(keyword string, offset, limit int) ([]model.Topic, int64, error)
	DeleteTopic(id string) error
}

type momentRepository struct {
	db *gorm.DB
}

func NewMomentRepository(db *gorm.DB) MomentRepository {
	return &momentRepository{db: db}
}

// --- Post ---

func (r *momentRepository) CreatePost(post *model.Post) error {
	return r.db.Create(post).Error
}

func (r *momentRepository) GetPostByID(id string) (*model.Post, error) {
	var post model.Post
	if err := r.db.Preload("Topics").Where("id = ?", id).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *momentRepository) GetPosts(status string, offset, limit int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := r.db.Model(&model.Post{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Topics").Order("created_at desc").Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *momentRepository) UpdatePostStatus(id string, status string) error {
	return r.db.Model(&model.Post{}).Where("id = ?", id).Update("status", status).Error
}

// --- Comment ---

func (r *momentRepository) CreateComment(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

func (r *momentRepository) GetCommentByID(id string) (*model.Comment, error) {
	var comment model.Comment
	if err := r.db.Where("id = ?", id).First(&comment).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *momentRepository) GetCommentsByPostID(postID string, offset, limit int) ([]model.Comment, int64, error) {
	var comments []model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).Where("post_id = ?", postID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at asc").Offset(offset).Limit(limit).Find(&comments).Error; err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

// --- Like ---

func (r *momentRepository) CreateLike(like *model.Like) error {
	return r.db.Create(like).Error
}

func (r *momentRepository) DeleteLike(userID, targetID string, targetType string) error {
	return r.db.Where("user_id = ? AND target_id = ? AND target_type = ?", userID, targetID, targetType).Delete(&model.Like{}).Error
}

func (r *momentRepository) HasLiked(userID, targetID string, targetType string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Like{}).Where("user_id = ? AND target_id = ? AND target_type = ?", userID, targetID, targetType).Count(&count).Error
	return count > 0, err
}

// --- Topic ---

func (r *momentRepository) GetTopicByName(name string) (*model.Topic, error) {
	var topic model.Topic
	if err := r.db.Where("name = ?", name).First(&topic).Error; err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *momentRepository) CreateTopic(topic *model.Topic) error {
	return r.db.Create(topic).Error
}

func (r *momentRepository) GetTopics(keyword string, offset, limit int) ([]model.Topic, int64, error) {
	var topics []model.Topic
	var total int64

	query := r.db.Model(&model.Topic{})
	if keyword != "" {
		query = query.Where("name ILIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&topics).Error; err != nil {
		return nil, 0, err
	}
	return topics, total, nil
}

func (r *momentRepository) DeleteTopic(id string) error {
	return r.db.Where("id = ?", id).Delete(&model.Topic{}).Error
}
