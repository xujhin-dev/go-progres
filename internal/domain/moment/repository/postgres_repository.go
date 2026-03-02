package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/pkg/database"
	baseModel "user_crud_jwt/pkg/model"

	"github.com/lib/pq"
)

// PostgresMomentRepository PostgreSQL实现的动态仓库
type PostgresMomentRepository struct {
	db *database.DB
}

// NewPostgresMomentRepository 创建PostgreSQL动态仓库
func NewPostgresMomentRepository(db *database.DB) MomentRepository {
	return &PostgresMomentRepository{db: db}
}

func (r *PostgresMomentRepository) CreatePost(ctx context.Context, post *model.Post) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	post.ID = base.ID
	post.CreatedAt = base.CreatedAt
	post.UpdatedAt = base.UpdatedAt

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. 插入动态
	_, err = tx.ExecContext(ctx, `
		INSERT INTO posts (id, user_id, content, media_urls, type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, post.ID, post.UserID, post.Content, post.MediaURLs, post.Type, post.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}

	// 2. 处理话题关联
	for _, topic := range post.Topics {
		// 检查话题是否存在
		var topicID string
		err = tx.QueryRowContext(ctx, `
			SELECT id FROM topics WHERE name = $1
		`, topic.Name).Scan(&topicID)

		if err == sql.ErrNoRows {
			// 创建新话题
			topicBase := baseModel.NewBaseModel()
			topic.ID = topicBase.ID
			topic.CreatedAt = topicBase.CreatedAt
			topic.UpdatedAt = topicBase.UpdatedAt

			_, err = tx.ExecContext(ctx, `
				INSERT INTO topics (id, name, created_at, updated_at)
				VALUES ($1, $2, $3, $4)
			`, topic.ID, topic.Name, topic.CreatedAt, topic.UpdatedAt)
			if err != nil {
				return fmt.Errorf("create topic: %w", err)
			}
			topicID = topic.ID
		} else if err != nil {
			return fmt.Errorf("query topic: %w", err)
		}

		// 创建动态-话题关联
		_, err = tx.ExecContext(ctx, `
			INSERT INTO post_topics (post_id, topic_id, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (post_id, topic_id) DO NOTHING
		`, post.ID, topicID, time.Now())
		if err != nil {
			return fmt.Errorf("create post-topic relation: %w", err)
		}
	}

	return tx.Commit()
}

func (r *PostgresMomentRepository) GetPostByID(ctx context.Context, id string) (*model.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.content, p.media_urls, p.type, p.status, 
		       p.created_at, p.updated_at, p.deleted_at,
		       COALESCE(array_agg(t.name), '{}') as topic_names
		FROM posts p
		LEFT JOIN post_topics pt ON p.id = pt.post_id
		LEFT JOIN topics t ON pt.topic_id = t.id
		WHERE p.id = $1 AND p.deleted_at IS NULL
		GROUP BY p.id, p.user_id, p.content, p.media_urls, p.type, p.status, p.created_at, p.updated_at, p.deleted_at
	`

	var postID, userID, content string
	var mediaURLs []byte
	var postType, status string
	var createdAt, updatedAt time.Time
	var deletedAt *time.Time
	var topicNames []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&postID, &userID, &content, &mediaURLs, &postType, &status,
		&createdAt, &updatedAt, &deletedAt,
		pq.Array(&topicNames),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("post not found")
		}
		return nil, fmt.Errorf("query post: %w", err)
	}

	post := &model.Post{
		BaseModel: baseModel.BaseModel{
			ID:        postID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: deletedAt,
		},
		UserID:    userID,
		Content:   content,
		MediaURLs: mediaURLs,
		Type:      postType,
		Status:    status,
	}

	// 转换话题
	for _, topicName := range topicNames {
		if topicName != "" {
			post.Topics = append(post.Topics, model.Topic{
				BaseModel: baseModel.BaseModel{},
				Name:      topicName,
			})
		}
	}

	return post, nil
}

func (r *PostgresMomentRepository) GetPosts(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.content, p.media_urls, p.type, p.status, 
		       p.created_at, p.updated_at, p.deleted_at,
		       COALESCE(array_agg(t.name), '{}') as topic_names
		FROM posts p
		LEFT JOIN post_topics pt ON p.id = pt.post_id
		LEFT JOIN topics t ON pt.topic_id = t.id
		WHERE p.deleted_at IS NULL AND p.status = 'approved'
		GROUP BY p.id, p.user_id, p.content, p.media_urls, p.type, p.status, p.created_at, p.updated_at, p.deleted_at
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var postID, userID, content string
		var mediaURLs []byte
		var postType, status string
		var createdAt, updatedAt time.Time
		var deletedAt *time.Time
		var topicNames []string

		err := rows.Scan(
			&postID, &userID, &content, &mediaURLs, &postType, &status,
			&createdAt, &updatedAt, &deletedAt,
			pq.Array(&topicNames),
		)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}

		post := &model.Post{
			BaseModel: baseModel.BaseModel{
				ID:        postID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			},
			UserID:    userID,
			Content:   content,
			MediaURLs: mediaURLs,
			Type:      postType,
			Status:    status,
		}

		// 转换话题
		for _, topicName := range topicNames {
			if topicName != "" {
				post.Topics = append(post.Topics, model.Topic{
					BaseModel: baseModel.BaseModel{},
					Name:      topicName,
				})
			}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *PostgresMomentRepository) GetPostsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.content, p.media_urls, p.type, p.status, 
		       p.created_at, p.updated_at, p.deleted_at,
		       COALESCE(array_agg(t.name), '{}') as topic_names
		FROM posts p
		LEFT JOIN post_topics pt ON p.id = pt.post_id
		LEFT JOIN topics t ON pt.topic_id = t.id
		WHERE p.deleted_at IS NULL AND p.user_id = $1
		GROUP BY p.id, p.user_id, p.content, p.media_urls, p.type, p.status, p.created_at, p.updated_at, p.deleted_at
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query user posts: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var postID, user_id, content string
		var mediaURLs []byte
		var postType, status string
		var createdAt, updatedAt time.Time
		var deletedAt *time.Time
		var topicNames []string

		err := rows.Scan(
			&postID, &user_id, &content, &mediaURLs, &postType, &status,
			&createdAt, &updatedAt, &deletedAt,
			pq.Array(&topicNames),
		)
		if err != nil {
			return nil, fmt.Errorf("scan user post: %w", err)
		}

		post := &model.Post{
			BaseModel: baseModel.BaseModel{
				ID:        postID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			},
			UserID:    user_id,
			Content:   content,
			MediaURLs: mediaURLs,
			Type:      postType,
			Status:    status,
		}

		// 转换话题
		for _, topicName := range topicNames {
			if topicName != "" {
				post.Topics = append(post.Topics, model.Topic{
					BaseModel: baseModel.BaseModel{},
					Name:      topicName,
				})
			}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *PostgresMomentRepository) UpdatePost(ctx context.Context, post *model.Post) error {
	query := `
		UPDATE posts 
		SET content = $1, media_urls = $2, type = $3, status = $4
		WHERE id = $5 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query,
		post.Content, post.MediaURLs, post.Type, post.Status, post.ID)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) UpdatePostStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE posts 
		SET status = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update post status: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) DeletePost(ctx context.Context, id string) error {
	query := `
		UPDATE posts 
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) CreateTopic(ctx context.Context, topic *model.Topic) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	topic.ID = base.ID
	topic.CreatedAt = base.CreatedAt
	topic.UpdatedAt = base.UpdatedAt

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO topics (id, name, created_at)
		VALUES ($1, $2, $3)
	`, topic.ID, topic.Name, topic.CreatedAt)
	if err != nil {
		return fmt.Errorf("create topic: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) GetTopicByName(ctx context.Context, name string) (*model.Topic, error) {
	var topicID, topicName string
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, created_at FROM topics WHERE name = $1 AND deleted_at IS NULL
	`, name).Scan(&topicID, &topicName, &createdAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("topic not found")
		}
		return nil, fmt.Errorf("query topic: %w", err)
	}

	return &model.Topic{
		BaseModel: baseModel.BaseModel{
			ID:        topicID,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		Name: topicName,
	}, nil
}

func (r *PostgresMomentRepository) GetTopics(ctx context.Context, limit, offset int) ([]*model.Topic, error) {
	query := `
		SELECT id, name, created_at 
		FROM topics 
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query topics: %w", err)
	}
	defer rows.Close()

	var topics []*model.Topic
	for rows.Next() {
		var topicID, topicName string
		var createdAt time.Time

		err := rows.Scan(&topicID, &topicName, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}

		topics = append(topics, &model.Topic{
			BaseModel: baseModel.BaseModel{
				ID:        topicID,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			},
			Name: topicName,
		})
	}

	return topics, nil
}

func (r *PostgresMomentRepository) GetTopicByID(ctx context.Context, id string) (*model.Topic, error) {
	var topicID, topicName string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, created_at, updated_at FROM topics WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&topicID, &topicName, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("topic not found")
		}
		return nil, fmt.Errorf("query topic by id: %w", err)
	}

	return &model.Topic{
		BaseModel: baseModel.BaseModel{
			ID:        topicID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: topicName,
	}, nil
}

func (r *PostgresMomentRepository) DeleteTopic(ctx context.Context, id string) error {
	query := `UPDATE topics SET deleted_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("delete topic: %w", err)
	}
	return nil
}

func (r *PostgresMomentRepository) GetTopicsByName(ctx context.Context, name string) ([]*model.Topic, error) {
	query := `
		SELECT id, name, created_at, updated_at 
		FROM topics 
		WHERE name ILIKE $1 || '%' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, name)
	if err != nil {
		return nil, fmt.Errorf("query topics by name: %w", err)
	}
	defer rows.Close()

	var topics []*model.Topic
	for rows.Next() {
		var topicID, topicName string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&topicID, &topicName, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}

		topics = append(topics, &model.Topic{
			BaseModel: baseModel.BaseModel{
				ID:        topicID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			Name: topicName,
		})
	}

	return topics, nil
}

func (r *PostgresMomentRepository) AssociatePostWithTopic(ctx context.Context, postID, topicID string) error {
	query := `
		INSERT INTO post_topics (post_id, topic_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id, topic_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, postID, topicID, time.Now())
	if err != nil {
		return fmt.Errorf("associate post with topic: %w", err)
	}

	return nil
}

// 评论相关方法
func (r *PostgresMomentRepository) CreateComment(ctx context.Context, comment *model.Comment) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	comment.ID = base.ID
	comment.CreatedAt = base.CreatedAt
	comment.UpdatedAt = base.UpdatedAt

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO comments (id, post_id, user_id, content, parent_id, root_id, level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, comment.ID, comment.PostID, comment.UserID, comment.Content, comment.ParentID, comment.RootID, comment.Level, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) GetCommentByID(ctx context.Context, id string) (*model.Comment, error) {
	var commentID, postID, userID, content string
	var parentID, rootID *string
	var level int
	var createdAt, updatedAt time.Time
	var deletedAt *time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, post_id, user_id, content, parent_id, root_id, level, created_at, updated_at, deleted_at
		FROM comments 
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&commentID, &postID, &userID, &content, &parentID, &rootID, &level, &createdAt, &updatedAt, &deletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("comment not found")
		}
		return nil, fmt.Errorf("query comment: %w", err)
	}

	comment := &model.Comment{
		BaseModel: baseModel.BaseModel{
			ID:        commentID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: deletedAt,
		},
		PostID:   postID,
		UserID:   userID,
		Content:  content,
		ParentID: "",
		RootID:   "",
		Level:    level,
	}

	if parentID != nil {
		comment.ParentID = *parentID
	}
	if rootID != nil {
		comment.RootID = *rootID
	}

	return comment, nil
}

func (r *PostgresMomentRepository) GetCommentsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, parent_id, root_id, level, created_at, updated_at, deleted_at
		FROM comments 
		WHERE post_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		var commentID, post_id, userID, content string
		var parentID, rootID *string
		var level int
		var createdAt, updatedAt time.Time
		var deletedAt *time.Time

		err := rows.Scan(&commentID, &post_id, &userID, &content, &parentID, &rootID, &level, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}

		comment := &model.Comment{
			BaseModel: baseModel.BaseModel{
				ID:        commentID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			},
			PostID:   post_id,
			UserID:   userID,
			Content:  content,
			ParentID: "",
			RootID:   "",
			Level:    level,
		}

		if parentID != nil {
			comment.ParentID = *parentID
		}
		if rootID != nil {
			comment.RootID = *rootID
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (r *PostgresMomentRepository) UpdateComment(ctx context.Context, comment *model.Comment) error {
	query := `
		UPDATE comments 
		SET content = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, comment.Content, comment.UpdatedAt, comment.ID)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) DeleteComment(ctx context.Context, id string) error {
	query := `
		UPDATE comments 
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}

	return nil
}

// 点赞相关方法
func (r *PostgresMomentRepository) CreateLike(ctx context.Context, like *model.Like) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	like.ID = base.ID
	like.CreatedAt = base.CreatedAt

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO likes (id, user_id, target_id, target_type, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, like.ID, like.UserID, like.TargetID, like.TargetType, like.CreatedAt)
	if err != nil {
		return fmt.Errorf("create like: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) DeleteLike(ctx context.Context, userID, targetID string, targetType string) error {
	query := `
		DELETE FROM likes 
		WHERE user_id = $1 AND target_id = $2 AND target_type = $3
	`

	_, err := r.db.ExecContext(ctx, query, userID, targetID, targetType)
	if err != nil {
		return fmt.Errorf("delete like: %w", err)
	}

	return nil
}

func (r *PostgresMomentRepository) GetLikesByTarget(ctx context.Context, targetID string, targetType string, limit, offset int) ([]*model.Like, error) {
	query := `
		SELECT id, user_id, target_id, target_type, created_at
		FROM likes 
		WHERE target_id = $1 AND target_type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, targetID, targetType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query likes: %w", err)
	}
	defer rows.Close()

	var likes []*model.Like
	for rows.Next() {
		var likeID, userID, target_id, target_type string
		var createdAt time.Time

		err := rows.Scan(&likeID, &userID, &target_id, &target_type, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan like: %w", err)
		}

		likes = append(likes, &model.Like{
			BaseModel: baseModel.BaseModel{
				ID:        likeID,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			},
			UserID:     userID,
			TargetID:   target_id,
			TargetType: target_type,
		})
	}

	return likes, nil
}

func (r *PostgresMomentRepository) HasLiked(ctx context.Context, userID, targetID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM likes 
		WHERE user_id = $1 AND target_id = $2 AND deleted_at IS NULL
	`, userID, targetID).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("check like exists: %w", err)
	}

	return count > 0, nil
}
