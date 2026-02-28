package service

import (
	"context"
	"encoding/json"
	"errors"
	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/internal/domain/moment/repository"
)

type MomentService interface {
	PublishPost(userID string, content string, mediaURLs []string, postType string, topicNames []string) (*model.Post, error)
	AuditPost(postID string, status string) error
	GetFeed(page, limit int) ([]model.Post, int64, error)         // Get approved posts
	GetPendingPosts(page, limit int) ([]model.Post, int64, error) // For admin

	AddComment(userID, postID string, content string, parentID string) (*model.Comment, error)
	GetPostComments(postID string, page, limit int) ([]model.Comment, int64, error)

	ToggleLike(userID, targetID string, targetType string) (bool, error) // Returns true if liked, false if unliked

	GetTopicList(keyword string, page, limit int) ([]model.Topic, int64, error)
	DeleteTopic(id string) error
}

type momentService struct {
	repo repository.MomentRepository
}

func NewMomentService(repo repository.MomentRepository) MomentService {
	return &momentService{repo: repo}
}

func (s *momentService) PublishPost(userID string, content string, mediaURLs []string, postType string, topicNames []string) (*model.Post, error) {
	// 1. Prepare Media JSON
	mediaJSON, _ := json.Marshal(mediaURLs)

	// 2. Handle Topics
	var topics []model.Topic
	for _, name := range topicNames {
		topic, err := s.repo.GetTopicByName(context.Background(), name)
		if err != nil {
			if err.Error() == "topic not found" {
				topic = &model.Topic{Name: name}
				if err := s.repo.CreateTopic(context.Background(), topic); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		topics = append(topics, *topic)
	}

	post := &model.Post{
		UserID:    userID,
		Content:   content,
		MediaURLs: mediaJSON,
		Type:      postType,
		Status:    "pending", // Default to pending
		Topics:    topics,
	}

	if err := s.repo.CreatePost(context.Background(), post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *momentService) AuditPost(postID string, status string) error {
	if status != "approved" && status != "rejected" {
		return errors.New("invalid status")
	}
	return s.repo.UpdatePostStatus(context.Background(), postID, status)
}

func (s *momentService) GetFeed(page, limit int) ([]model.Post, int64, error) {
	offset := (page - 1) * limit
	posts, err := s.repo.GetPosts(context.Background(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	// TODO: 获取总数，这里暂时返回 len(posts)
	var result []model.Post
	for _, p := range posts {
		result = append(result, *p)
	}
	return result, int64(len(result)), nil
}

func (s *momentService) GetPendingPosts(page, limit int) ([]model.Post, int64, error) {
	offset := (page - 1) * limit
	posts, err := s.repo.GetPosts(context.Background(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	// TODO: 过滤 pending 状态的动态，这里暂时返回所有
	var result []model.Post
	for _, p := range posts {
		result = append(result, *p)
	}
	return result, int64(len(result)), nil
}

func (s *momentService) AddComment(userID, postID string, content string, parentID string) (*model.Comment, error) {
	// Check if post exists and is approved
	post, err := s.repo.GetPostByID(context.Background(), postID)
	if err != nil {
		return nil, err
	}
	if post.Status != "approved" {
		return nil, errors.New("cannot comment on unapproved post")
	}

	comment := &model.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
		Level:   1, // 默认一级评论
	}

	// 处理回复逻辑
	if parentID != "" {
		// 获取父评论
		parentComment, err := s.repo.GetCommentByID(context.Background(), parentID)
		if err != nil {
			return nil, errors.New("parent comment not found")
		}

		// 验证父评论属于同一个帖子
		if parentComment.PostID != postID {
			return nil, errors.New("parent comment does not belong to this post")
		}

		comment.ParentID = parentID

		// 确定 RootID 和 Level
		if parentComment.Level == 1 {
			// 回复一级评论，这是二级评论
			comment.RootID = parentComment.ID
			comment.Level = 2
		} else {
			// 回复二级评论，仍然是二级评论（限制最多两层）
			comment.RootID = parentComment.RootID
			comment.Level = 2
		}
	}

	if err := s.repo.CreateComment(context.Background(), comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *momentService) GetPostComments(postID string, page, limit int) ([]model.Comment, int64, error) {
	offset := (page - 1) * limit
	comments, err := s.repo.GetCommentsByPostID(context.Background(), postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	var result []model.Comment
	for _, c := range comments {
		result = append(result, *c)
	}
	return result, int64(len(result)), nil
}

func (s *momentService) ToggleLike(userID, targetID string, targetType string) (bool, error) {
	liked, err := s.repo.HasLiked(context.Background(), userID, targetID)
	if err != nil {
		return false, err
	}

	if liked {
		// Unlike
		err := s.repo.DeleteLike(context.Background(), userID, targetID, targetType)
		return false, err
	} else {
		// Like
		like := &model.Like{
			UserID:     userID,
			TargetID:   targetID,
			TargetType: targetType,
		}
		err := s.repo.CreateLike(context.Background(), like)
		return true, err
	}
}

func (s *momentService) GetTopicList(keyword string, page, limit int) ([]model.Topic, int64, error) {
	offset := (page - 1) * limit
	topics, err := s.repo.GetTopics(context.Background(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	// TODO: 根据 keyword 过滤，这里暂时返回所有
	var result []model.Topic
	for _, t := range topics {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (s *momentService) DeleteTopic(id string) error {
	return s.repo.DeleteTopic(context.Background(), id)
}
