package service

import (
	"context"
	"fmt"
	"strings"

	"user_crud_jwt/internal/domain/moment/model"
	"user_crud_jwt/internal/domain/moment/repository"
)

// SearchService 搜索服务
type SearchService struct {
	repo repository.MomentRepository
}

// NewSearchService 创建搜索服务
func NewSearchService(repo repository.MomentRepository) *SearchService {
	return &SearchService{repo: repo}
}

// SearchResult 搜索结果
type SearchResult struct {
	Posts []model.Post `json:"posts"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword string `json:"keyword" binding:"required"`
	Type    string `json:"type"`   // text, image, video, all
	UserID  string `json:"userId"` // 按用户搜索
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
}

// SearchMoments 搜索动态
func (s *SearchService) SearchMoments(req SearchRequest) (*SearchResult, error) {
	// 获取所有动态（简化实现，实际应该有专门的搜索SQL）
	posts, err := s.repo.GetPosts(context.Background(), req.Limit, (req.Page-1)*req.Limit)
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}

	// 过滤搜索结果
	var filteredPosts []model.Post
	for _, post := range posts {
		if s.matchesSearchCriteria(*post, req) {
			filteredPosts = append(filteredPosts, *post)
		}
	}

	return &SearchResult{
		Posts: filteredPosts,
		Total: int64(len(filteredPosts)),
		Page:  req.Page,
		Limit: req.Limit,
	}, nil
}

// SearchTopics 搜索话题
func (s *SearchService) SearchTopics(keyword string, limit int) ([]model.Topic, error) {
	topics, err := s.repo.GetTopicsByName(context.Background(), keyword)
	if err != nil {
		return nil, fmt.Errorf("search topics: %w", err)
	}

	// 限制结果数量
	if len(topics) > limit {
		topics = topics[:limit]
	}

	// 转换为切片
	result := make([]model.Topic, len(topics))
	for i, topic := range topics {
		result[i] = *topic
	}

	return result, nil
}

// matchesSearchCriteria 检查动态是否匹配搜索条件
func (s *SearchService) matchesSearchCriteria(post model.Post, req SearchRequest) bool {
	// 关键词搜索
	if req.Keyword != "" {
		keyword := strings.ToLower(req.Keyword)
		content := strings.ToLower(post.Content)

		if !strings.Contains(content, keyword) {
			// 检查话题
			topicMatch := false
			for _, topic := range post.Topics {
				if strings.Contains(strings.ToLower(topic.Name), keyword) {
					topicMatch = true
					break
				}
			}
			if !topicMatch {
				return false
			}
		}
	}

	// 类型过滤
	if req.Type != "" && req.Type != "all" {
		if post.Type != req.Type {
			return false
		}
	}

	// 用户过滤
	if req.UserID != "" {
		if post.UserID != req.UserID {
			return false
		}
	}

	return true
}

// GetHotTopics 获取热门话题
func (s *SearchService) GetHotTopics(limit int) ([]model.Topic, error) {
	// 简化实现：获取所有话题并按名称排序
	topics, err := s.repo.GetTopics(context.Background(), limit, 0)
	if err != nil {
		return nil, fmt.Errorf("get hot topics: %w", err)
	}

	// 转换为切片
	result := make([]model.Topic, len(topics))
	for i, topic := range topics {
		result[i] = *topic
	}

	return result, nil
}

// GetUserMoments 获取用户动态
func (s *SearchService) GetUserMoments(userID string, page, limit int) (*SearchResult, error) {
	posts, err := s.repo.GetPostsByUserID(context.Background(), userID, limit, (page-1)*limit)
	if err != nil {
		return nil, fmt.Errorf("get user moments: %w", err)
	}

	// 转换为切片
	result := make([]model.Post, len(posts))
	for i, post := range posts {
		result[i] = *post
	}

	return &SearchResult{
		Posts: result,
		Total: int64(len(result)),
		Page:  page,
		Limit: limit,
	}, nil
}
