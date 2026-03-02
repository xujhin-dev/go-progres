package repository

import (
	"context"
	"errors"
	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/pkg/database"
	baseModel "user_crud_jwt/pkg/model"
)

// SimpleMomentRepository 简单的动态仓库实现
type SimpleMomentRepository struct {
	db     *database.DB
	posts  []*model.Post   // 内存存储，仅用于测试
	topics []*model.Topic  // 内存存储，仅用于测试
	comments []*model.Comment // 内存存储，仅用于测试
	likes   []*model.Like    // 内存存储，仅用于测试
}

// NewSimpleMomentRepository 创建简单动态仓库
func NewSimpleMomentRepository(db *database.DB) MomentRepository {
	return &SimpleMomentRepository{db: db}
}

func (r *SimpleMomentRepository) CreatePost(ctx context.Context, post *model.Post) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	post.ID = base.ID
	post.CreatedAt = base.CreatedAt
	post.UpdatedAt = base.UpdatedAt
	
	// 存储到内存中（仅用于测试）
	r.posts = append(r.posts, post)
	
	return nil
}

func (r *SimpleMomentRepository) GetPostByID(ctx context.Context, id string) (*model.Post, error) {
	for _, post := range r.posts {
		if post.ID == id {
			return post, nil
		}
	}
	return nil, errors.New("post not found")
}

func (r *SimpleMomentRepository) GetPosts(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	// 返回最新的posts（仅用于测试）
	if offset >= len(r.posts) {
		return []*model.Post{}, nil
	}
	
	end := offset + limit
	if end > len(r.posts) {
		end = len(r.posts)
	}
	
	result := make([]*model.Post, 0, end-offset)
	for i := len(r.posts) - 1 - offset; i >= len(r.posts)-end && i >= 0; i-- {
		result = append(result, r.posts[i])
	}
	
	return result, nil
}

func (r *SimpleMomentRepository) GetPostsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Post, error) {
	// 返回指定用户的posts（仅用于测试）
	var userPosts []*model.Post
	for _, post := range r.posts {
		if post.UserID == userID {
			userPosts = append(userPosts, post)
		}
	}
	
	if offset >= len(userPosts) {
		return []*model.Post{}, nil
	}
	
	end := offset + limit
	if end > len(userPosts) {
		end = len(userPosts)
	}
	
	result := make([]*model.Post, 0, end-offset)
	for i := offset; i < end; i++ {
		result = append(result, userPosts[i])
	}
	
	return result, nil
}

func (r *SimpleMomentRepository) UpdatePost(ctx context.Context, post *model.Post) error {
	// 简单实现：更新内存中的动态（仅用于测试）
	for i, p := range r.posts {
		if p.ID == post.ID {
			r.posts[i] = post
			return nil
		}
	}
	return errors.New("post not found")
}

func (r *SimpleMomentRepository) UpdatePostStatus(ctx context.Context, id string, status string) error {
	// 简单实现：更新内存中的动态状态（仅用于测试）
	for i, post := range r.posts {
		if post.ID == id {
			r.posts[i].Status = status
			return nil
		}
	}
	return errors.New("post not found")
}

func (r *SimpleMomentRepository) DeletePost(ctx context.Context, id string) error {
	// 简单实现：从内存中删除动态（仅用于测试）
	for i, post := range r.posts {
		if post.ID == id {
			r.posts = append(r.posts[:i], r.posts[i+1:]...)
			return nil
		}
	}
	return errors.New("post not found")
}

func (r *SimpleMomentRepository) CreateTopic(ctx context.Context, topic *model.Topic) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	topic.ID = base.ID
	topic.CreatedAt = base.CreatedAt
	topic.UpdatedAt = base.UpdatedAt
	
	// 存储到内存中（仅用于测试）
	r.topics = append(r.topics, topic)
	
	return nil
}

func (r *SimpleMomentRepository) GetTopicByName(ctx context.Context, name string) (*model.Topic, error) {
	for _, topic := range r.topics {
		if topic.Name == name {
			return topic, nil
		}
	}
	return nil, errors.New("topic not found")
}

func (r *SimpleMomentRepository) GetTopicsByName(ctx context.Context, name string) ([]*model.Topic, error) {
	// 简单实现：在内存中搜索话题（仅用于测试）
	var result []*model.Topic
	for _, topic := range r.topics {
		if len(topic.Name) >= len(name) && topic.Name[:len(name)] == name {
			result = append(result, topic)
		}
	}
	return result, nil
}

func (r *SimpleMomentRepository) GetTopics(ctx context.Context, limit, offset int) ([]*model.Topic, error) {
	// 返回最新的topics（仅用于测试）
	if offset >= len(r.topics) {
		return []*model.Topic{}, nil
	}
	
	end := offset + limit
	if end > len(r.topics) {
		end = len(r.topics)
	}
	
	result := make([]*model.Topic, 0, end-offset)
	for i := len(r.topics) - 1 - offset; i >= len(r.topics)-end && i >= 0; i-- {
		result = append(result, r.topics[i])
	}
	
	return result, nil
}

func (r *SimpleMomentRepository) GetTopicByID(ctx context.Context, id string) (*model.Topic, error) {
	for _, topic := range r.topics {
		if topic.ID == id {
			return topic, nil
		}
	}
	return nil, errors.New("topic not found")
}

func (r *SimpleMomentRepository) DeleteTopic(ctx context.Context, id string) error {
	// 简单实现：从内存中删除话题（仅用于测试）
	for i, topic := range r.topics {
		if topic.ID == id {
			r.topics = append(r.topics[:i], r.topics[i+1:]...)
			return nil
		}
	}
	return errors.New("topic not found")
}

func (r *SimpleMomentRepository) AssociatePostWithTopic(ctx context.Context, postID, topicID string) error {
	// 简单实现：在内存中关联（仅用于测试）
	return nil
}

// 评论相关方法实现
func (r *SimpleMomentRepository) CreateComment(ctx context.Context, comment *model.Comment) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	comment.ID = base.ID
	comment.CreatedAt = base.CreatedAt
	comment.UpdatedAt = base.UpdatedAt
	
	// 简单实现：存储到内存中（仅用于测试）
	if r.comments == nil {
		r.comments = make([]*model.Comment, 0)
	}
	r.comments = append(r.comments, comment)
	
	return nil
}

func (r *SimpleMomentRepository) GetCommentByID(ctx context.Context, id string) (*model.Comment, error) {
	if r.comments == nil {
		return nil, errors.New("comment not found")
	}
	
	for _, comment := range r.comments {
		if comment.ID == id {
			return comment, nil
		}
	}
	return nil, errors.New("comment not found")
}

func (r *SimpleMomentRepository) GetCommentsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.Comment, error) {
	if r.comments == nil {
		return []*model.Comment{}, nil
	}
	
	var result []*model.Comment
	count := 0
	for _, comment := range r.comments {
		if comment.PostID == postID {
			if count >= offset && len(result) < limit {
				result = append(result, comment)
			}
			count++
		}
	}
	
	return result, nil
}

func (r *SimpleMomentRepository) UpdateComment(ctx context.Context, comment *model.Comment) error {
	// 简单实现：更新内存中的评论（仅用于测试）
	for i, c := range r.comments {
		if c.ID == comment.ID {
			r.comments[i] = comment
			return nil
		}
	}
	return errors.New("comment not found")
}

func (r *SimpleMomentRepository) DeleteComment(ctx context.Context, id string) error {
	// 简单实现：从内存中删除评论（仅用于测试）
	for i, comment := range r.comments {
		if comment.ID == id {
			r.comments = append(r.comments[:i], r.comments[i+1:]...)
			return nil
		}
	}
	return errors.New("comment not found")
}

// 点赞相关方法实现
func (r *SimpleMomentRepository) CreateLike(ctx context.Context, like *model.Like) error {
	// 使用BaseModel生成ID和时间戳
	base := baseModel.NewBaseModel()
	like.ID = base.ID
	like.CreatedAt = base.CreatedAt
	
	// 简单实现：存储到内存中（仅用于测试）
	if r.likes == nil {
		r.likes = make([]*model.Like, 0)
	}
	r.likes = append(r.likes, like)
	
	return nil
}

func (r *SimpleMomentRepository) DeleteLike(ctx context.Context, userID, targetID string, targetType string) error {
	// 简单实现：从内存中删除点赞（仅用于测试）
	for i, like := range r.likes {
		if like.UserID == userID && like.TargetID == targetID && like.TargetType == targetType {
			r.likes = append(r.likes[:i], r.likes[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *SimpleMomentRepository) GetLikesByTarget(ctx context.Context, targetID string, targetType string, limit, offset int) ([]*model.Like, error) {
	if r.likes == nil {
		return []*model.Like{}, nil
	}
	
	var result []*model.Like
	count := 0
	for _, like := range r.likes {
		if like.TargetID == targetID && like.TargetType == targetType {
			if count >= offset && len(result) < limit {
				result = append(result, like)
			}
			count++
		}
	}
	
	return result, nil
}

func (r *SimpleMomentRepository) HasLiked(ctx context.Context, userID, targetID string) (bool, error) {
	if r.likes == nil {
		return false, nil
	}
	
	for _, like := range r.likes {
		if like.UserID == userID && like.TargetID == targetID {
			return true, nil
		}
	}
	return false, nil
}
