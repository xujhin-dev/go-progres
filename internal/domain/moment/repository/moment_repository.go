package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/pkg/database"
)

// MomentXRepository 使用 SQLX 实现的时刻仓库
type MomentXRepository struct {
	db *database.DB
}

// NewMomentRepository 创建新的时刻仓库
func NewMomentRepository(db *database.DB) MomentRepository {
	return &MomentXRepository{db: db}
}

// CreatePost 创建帖子
func (r *MomentXRepository) CreatePost(post *model.Post) error {
	query := `
		INSERT INTO posts (
			id, created_at, updated_at, user_id, content, media_urls, type, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		post.ID, post.CreatedAt, post.UpdatedAt, post.UserID,
		post.Content, post.MediaURLs, post.Type, post.Status,
	)

	return err
}

// GetPostByID 根据 ID 获取帖子
func (r *MomentXRepository) GetPostByID(id string) (*model.Post, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, user_id, content, media_urls, type, status, like_count, comment_count
		SELECT id::text, created_at, updated_at, deleted_at, user_id, content, status, like_count, comment_count
		FROM posts 
		WHERE id = $1 AND deleted_at IS NULL`

	var post model.Post
	err := r.db.GetContext(context.Background(), &post, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, err
	}

	return &post, nil
}

// GetPosts 获取帖子列表
func (r *MomentXRepository) GetPosts(status string, offset, limit int) ([]model.Post, int64, error) {
	// 获取总数
	countQuery := `SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL AND status = $1`
	var total int64
	err := r.db.GetContext(context.Background(), &total, countQuery, status)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	listQuery := `
		SELECT id::text, created_at, updated_at, deleted_at, user_id, content, status, like_count, comment_count
		FROM posts 
		WHERE deleted_at IS NULL AND status = $1
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	var posts []model.Post
	err = r.db.SelectContext(context.Background(), &posts, listQuery, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// UpdatePostStatus 更新帖子状态
func (r *MomentXRepository) UpdatePostStatus(id string, status string) error {
	query := `
		UPDATE posts 
		SET status = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(context.Background(), query, status, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// CreateComment 创建评论
func (r *MomentXRepository) CreateComment(comment *model.Comment) error {
	query := `
		INSERT INTO comments (
			id, created_at, updated_at, post_id, user_id, content
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		comment.ID, comment.CreatedAt, comment.UpdatedAt,
		comment.PostID, comment.UserID, comment.Content,
	)

	return err
}

// GetCommentByID 根据 ID 获取评论
func (r *MomentXRepository) GetCommentByID(id string) (*model.Comment, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, post_id, user_id, content
		FROM comments 
		WHERE id = $1 AND deleted_at IS NULL`

	var comment model.Comment
	err := r.db.GetContext(context.Background(), &comment, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("comment not found")
		}
		return nil, err
	}

	return &comment, nil
}

// GetCommentsByPostID 根据帖子ID获取评论列表
func (r *MomentXRepository) GetCommentsByPostID(postID string, offset, limit int) ([]model.Comment, int64, error) {
	// 获取总数
	countQuery := `SELECT COUNT(*) FROM comments WHERE deleted_at IS NULL AND post_id = $1`
	var total int64
	err := r.db.GetContext(context.Background(), &total, countQuery, postID)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	listQuery := `
		SELECT id::text, created_at, updated_at, deleted_at, post_id, user_id, content
		FROM comments 
		WHERE deleted_at IS NULL AND post_id = $1
		ORDER BY created_at ASC 
		LIMIT $2 OFFSET $3`

	var comments []model.Comment
	err = r.db.SelectContext(context.Background(), &comments, listQuery, postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// CreateLike 创建点赞
func (r *MomentXRepository) CreateLike(like *model.Like) error {
	query := `
		INSERT INTO likes (
			id, created_at, updated_at, user_id, target_id, target_type
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		like.ID, like.CreatedAt, like.UpdatedAt,
		like.UserID, like.TargetID, like.TargetType,
	)

	return err
}

// DeleteLike 删除点赞
func (r *MomentXRepository) DeleteLike(userID, targetID string, targetType string) error {
	query := `
		UPDATE likes 
		SET deleted_at = $1, updated_at = $2
		WHERE user_id = $3 AND target_id = $4 AND target_type = $5 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(context.Background(), query, time.Now(), time.Now(), userID, targetID, targetType)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("like not found")
	}

	return nil
}

// HasLiked 检查是否已点赞
func (r *MomentXRepository) HasLiked(userID, targetID string, targetType string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM likes 
		WHERE user_id = $1 AND target_id = $2 AND target_type = $3 AND deleted_at IS NULL`

	var count int64
	err := r.db.GetContext(context.Background(), &count, query, userID, targetID, targetType)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetTopicByName 根据名称获取话题
func (r *MomentXRepository) GetTopicByName(name string) (*model.Topic, error) {
	query := `
		SELECT id::text, created_at, updated_at, deleted_at, name
		FROM topics 
		WHERE name = $1 AND deleted_at IS NULL`

	var topic model.Topic
	err := r.db.GetContext(context.Background(), &topic, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("topic not found")
		}
		return nil, err
	}

	return &topic, nil
}

// CreateTopic 创建话题
func (r *MomentXRepository) CreateTopic(topic *model.Topic) error {
	query := `
		INSERT INTO topics (
			id, created_at, updated_at, name
		) VALUES (
			$1, $2, $3, $4
		)`

	_, err := r.db.ExecContext(context.Background(), query,
		topic.ID, topic.CreatedAt, topic.UpdatedAt, topic.Name,
	)

	return err
}

// GetTopics 获取话题列表
func (r *MomentXRepository) GetTopics(keyword string, offset, limit int) ([]model.Topic, int64, error) {
	// 获取总数
	countQuery := `SELECT COUNT(*) FROM topics WHERE deleted_at IS NULL AND name LIKE $1`
	var total int64
	err := r.db.GetContext(context.Background(), &total, countQuery, "%"+keyword+"%")
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	listQuery := `
		SELECT id::text, created_at, updated_at, deleted_at, name
		FROM topics 
		WHERE deleted_at IS NULL AND name LIKE $1
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	var topics []model.Topic
	err = r.db.SelectContext(context.Background(), &topics, listQuery, "%"+keyword+"%", limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return topics, total, nil
}

// DeleteTopic 删除话题
func (r *MomentXRepository) DeleteTopic(id string) error {
	query := `
		UPDATE topics 
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(context.Background(), query, time.Now(), time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("topic not found")
	}

	return nil
}
