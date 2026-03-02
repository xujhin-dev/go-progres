package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/internal/domain/moment/repository"
	"user_crud_jwt/pkg/cache"
)

// CachedMomentService 带缓存的动态服务
type CachedMomentService struct {
	repo  repository.MomentRepository
	cache *cache.MomentCache
}

// NewCachedMomentService 创建带缓存的动态服务
func NewCachedMomentService(repo repository.MomentRepository, cache *cache.MomentCache) MomentService {
	return &CachedMomentService{
		repo:  repo,
		cache: cache,
	}
}

func (s *CachedMomentService) PublishPost(userID string, content string, mediaURLs []string, postType string, topicNames []string) (*model.Post, error) {
	// 转换mediaURLs为JSON
	mediaJSON, _ := json.Marshal(mediaURLs)

	post := &model.Post{
		UserID:    userID,
		Content:   content,
		MediaURLs: mediaJSON,
		Type:      postType,
		Status:    "pending",
	}

	err := s.repo.CreatePost(context.Background(), post)
	if err != nil {
		return nil, err
	}

	// 清除相关缓存
	s.cache.InvalidateUserCache(context.Background(), userID)
	s.cache.InvalidateTopicCache(context.Background())

	return post, nil
}

func (s *CachedMomentService) AuditPost(postID string, status string) error {
	err := s.repo.UpdatePostStatus(context.Background(), postID, status)
	if err != nil {
		return err
	}

	// 清除相关缓存
	s.cache.InvalidateMomentCache(context.Background(), postID)

	return nil
}

func (s *CachedMomentService) GetFeed(page, limit int) ([]model.Post, int64, error) {
	// 尝试从缓存获取
	var cachedPosts []model.Post

	err := s.cache.GetMomentFeed(context.Background(), page, &cachedPosts)
	if err == nil {
		// 缓存命中
		return cachedPosts, int64(len(cachedPosts)), nil
	}

	// 缓存未命中，从数据库获取
	posts, err := s.repo.GetPosts(context.Background(), limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换为切片
	result := make([]model.Post, len(posts))
	for i, post := range posts {
		result[i] = *post
	}

	// 设置缓存
	s.cache.SetMomentFeed(context.Background(), page, result)

	return result, int64(len(result)), nil
}

func (s *CachedMomentService) GetPendingPosts(page, limit int) ([]model.Post, int64, error) {
	// 待审核动态不缓存，直接从数据库获取
	posts, err := s.repo.GetPosts(context.Background(), limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换为切片
	result := make([]model.Post, len(posts))
	for i, post := range posts {
		result[i] = *post
	}

	return result, int64(len(result)), nil
}

func (s *CachedMomentService) AddComment(userID, postID string, content string, parentID string) (*model.Comment, error) {
	// 检查动态权限
	post, err := s.repo.GetPostByID(context.Background(), postID)
	if err != nil {
		return nil, err
	}

	// 允许评论如果：1) 动态已审核，或 2) 用户评论自己的动态
	if post.Status != "approved" && post.UserID != userID {
		return nil, errors.New("cannot comment on unapproved post")
	}

	comment := &model.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
		Level:   1,
	}

	// 处理回复逻辑
	if parentID != "" {
		comment.ParentID = parentID
		// 查找父评论获取root_id
		parentComment, err := s.repo.GetCommentByID(context.Background(), parentID)
		if err == nil && parentComment != nil {
			comment.RootID = parentComment.RootID
			comment.Level = parentComment.Level + 1
		}
	} else {
		// 顶级评论，root_id指向自己
		comment.RootID = postID
	}

	err = s.repo.CreateComment(context.Background(), comment)
	if err != nil {
		return nil, err
	}

	// 清除相关缓存
	s.cache.InvalidateMomentCache(context.Background(), postID)

	return comment, nil
}

func (s *CachedMomentService) GetPostComments(postID string, page, limit int) ([]model.Comment, int64, error) {
	// 尝试从缓存获取
	commentCacheKey := fmt.Sprintf("comments:%s:page:%d", postID, page)
	var cachedComments []model.Comment

	err := s.cache.Get(context.Background(), commentCacheKey, &cachedComments)
	if err == nil {
		return cachedComments, int64(len(cachedComments)), nil
	}

	// 缓存未命中，从数据库获取
	comments, err := s.repo.GetCommentsByPostID(context.Background(), postID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换为切片
	result := make([]model.Comment, len(comments))
	for i, comment := range comments {
		result[i] = *comment
	}

	// 设置缓存
	s.cache.Set(context.Background(), commentCacheKey, result, 5*time.Minute)

	return result, int64(len(result)), nil
}

func (s *CachedMomentService) ToggleLike(userID, targetID string, targetType string) (bool, error) {
	// 检查是否已点赞
	hasLiked, err := s.repo.HasLiked(context.Background(), userID, targetID)
	if err != nil {
		return false, err
	}

	if hasLiked {
		// 取消点赞
		err = s.repo.DeleteLike(context.Background(), userID, targetID, targetType)
		if err != nil {
			return false, err
		}
		return false, nil
	} else {
		// 点赞
		like := &model.Like{
			UserID:     userID,
			TargetID:   targetID,
			TargetType: targetType,
		}

		err = s.repo.CreateLike(context.Background(), like)
		if err != nil {
			return false, err
		}
		return true, nil
	}
}

func (s *CachedMomentService) GetTopicList(keyword string, page, limit int) ([]model.Topic, int64, error) {
	// 尝试从缓存获取
	var cachedTopics []model.Topic

	err := s.cache.GetTopics(context.Background(), &cachedTopics)
	if err == nil {
		return cachedTopics, int64(len(cachedTopics)), nil
	}

	// 缓存未命中，从数据库获取
	topics, err := s.repo.GetTopics(context.Background(), limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换为切片
	result := make([]model.Topic, len(topics))
	for i, topic := range topics {
		result[i] = *topic
	}

	// 设置缓存
	s.cache.SetTopics(context.Background(), result)

	return result, int64(len(result)), nil
}

func (s *CachedMomentService) DeleteTopic(id string) error {
	err := s.repo.DeleteTopic(context.Background(), id)
	if err != nil {
		return err
	}

	// 清除话题缓存
	s.cache.InvalidateTopicCache(context.Background())

	return nil
}
