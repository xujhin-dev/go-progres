package service

import (
	"encoding/json"
	"errors"
	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/internal/domain/moment/repository"

	"gorm.io/gorm"
)

type MomentService interface {
	PublishPost(userID uint, content string, mediaURLs []string, postType string, topicNames []string) (*model.Post, error)
	AuditPost(postID uint, status string) error
	GetFeed(page, limit int) ([]model.Post, int64, error) // Get approved posts
	GetPendingPosts(page, limit int) ([]model.Post, int64, error) // For admin

	AddComment(userID, postID uint, content string, parentID uint) (*model.Comment, error)
	GetPostComments(postID uint, page, limit int) ([]model.Comment, int64, error)

	ToggleLike(userID, targetID uint, targetType string) (bool, error) // Returns true if liked, false if unliked

	GetTopicList(keyword string, page, limit int) ([]model.Topic, int64, error)
	DeleteTopic(id uint) error
}

type momentService struct {
	repo repository.MomentRepository
}

func NewMomentService(repo repository.MomentRepository) MomentService {
	return &momentService{repo: repo}
}

func (s *momentService) PublishPost(userID uint, content string, mediaURLs []string, postType string, topicNames []string) (*model.Post, error) {
	// 1. Prepare Media JSON
	mediaJSON, _ := json.Marshal(mediaURLs)

	// 2. Handle Topics
	var topics []model.Topic
	for _, name := range topicNames {
		topic, err := s.repo.GetTopicByName(name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				topic = &model.Topic{Name: name}
				if err := s.repo.CreateTopic(topic); err != nil {
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

	if err := s.repo.CreatePost(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *momentService) AuditPost(postID uint, status string) error {
	if status != "approved" && status != "rejected" {
		return errors.New("invalid status")
	}
	return s.repo.UpdatePostStatus(postID, status)
}

func (s *momentService) GetFeed(page, limit int) ([]model.Post, int64, error) {
	offset := (page - 1) * limit
	return s.repo.GetPosts("approved", offset, limit)
}

func (s *momentService) GetPendingPosts(page, limit int) ([]model.Post, int64, error) {
	offset := (page - 1) * limit
	return s.repo.GetPosts("pending", offset, limit)
}

func (s *momentService) AddComment(userID, postID uint, content string, parentID uint) (*model.Comment, error) {
	// Check if post exists and is approved
	post, err := s.repo.GetPostByID(postID)
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
	if parentID != 0 {
		// 获取父评论
		parentComment, err := s.repo.GetCommentByID(parentID)
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

	if err := s.repo.CreateComment(comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *momentService) GetPostComments(postID uint, page, limit int) ([]model.Comment, int64, error) {
	offset := (page - 1) * limit
	return s.repo.GetCommentsByPostID(postID, offset, limit)
}

func (s *momentService) ToggleLike(userID, targetID uint, targetType string) (bool, error) {
	liked, err := s.repo.HasLiked(userID, targetID, targetType)
	if err != nil {
		return false, err
	}

	if liked {
		// Unlike
		err := s.repo.DeleteLike(userID, targetID, targetType)
		return false, err
	} else {
		// Like
		like := &model.Like{
			UserID:     userID,
			TargetID:   targetID,
			TargetType: targetType,
		}
		err := s.repo.CreateLike(like)
		return true, err
	}
}

func (s *momentService) GetTopicList(keyword string, page, limit int) ([]model.Topic, int64, error) {
	offset := (page - 1) * limit
	return s.repo.GetTopics(keyword, offset, limit)
}

func (s *momentService) DeleteTopic(id uint) error {
	return s.repo.DeleteTopic(id)
}
