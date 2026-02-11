package handler

import (
	"net/http"
	"user_crud_jwt/internal/domain/moment/service"
	"user_crud_jwt/pkg/response"
	"user_crud_jwt/pkg/utils"

	"github.com/gin-gonic/gin"
)

type MomentHandler struct {
	service service.MomentService
}

func NewMomentHandler(s service.MomentService) *MomentHandler {
	return &MomentHandler{service: s}
}

// PublishInput 发布动态输入
type PublishInput struct {
	Content    string   `json:"content" binding:"required"`
	MediaURLs  []string `json:"mediaUrls"`
	Type       string   `json:"type" binding:"required,oneof=text image video"`
	TopicNames []string `json:"topics"`
}

// AuditInput 审核输入
type AuditInput struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`
}

// CommentInput 评论输入
type CommentInput struct {
	Content  string `json:"content" binding:"required"`
	ParentID string `json:"parentId"`
}

// LikeInput 点赞输入
type LikeInput struct {
	TargetID   string `json:"targetId" binding:"required"`
	TargetType string `json:"targetType" binding:"required,oneof=post comment"`
}

// PublishPost 发布动态
// @Summary 发布动态
// @Tags Moment
// @Accept json
// @Produce json
// @Param input body PublishInput true "动态内容"
// @Success 200 {object} model.Post
// @Router /moments/publish [post]
func (h *MomentHandler) PublishPost(c *gin.Context) {
	var input PublishInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	userID := getUserIdFromContext(c)
	post, err := h.service.PublishPost(userID, input.Content, input.MediaURLs, input.Type, input.TopicNames)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, post)
}

// AuditPost 审核动态 (管理员)
// @Summary 审核动态
// @Tags Moment
// @Accept json
// @Produce json
// @Param id path string true "动态ID"
// @Param input body AuditInput true "审核状态"
// @Success 200 {string} string "success"
// @Router /moments/{id}/audit [put]
func (h *MomentHandler) AuditPost(c *gin.Context) {
	id := c.Param("id")

	var input AuditInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	if err := h.service.AuditPost(id, input.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, "success")
}

// GetFeed 获取动态流
// @Summary 获取已审核动态
// @Tags Moment
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.PageResult
// @Router /moments/feed [get]
func (h *MomentHandler) GetFeed(c *gin.Context) {
	var p utils.Pagination
	c.ShouldBindQuery(&p)

	posts, total, err := h.service.GetFeed(p.Page, p.Limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, utils.PageResult{
		List:  posts,
		Total: total,
		Page:  p.Page,
		Limit: p.Limit,
	})
}

// AddComment 发表评论
func (h *MomentHandler) AddComment(c *gin.Context) {
	postID := c.Param("id")

	var input CommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	userID := getUserIdFromContext(c)
	comment, err := h.service.AddComment(userID, postID, input.Content, input.ParentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, comment)
}

// GetComments 获取评论列表
func (h *MomentHandler) GetComments(c *gin.Context) {
	postID := c.Param("id")

	var p utils.Pagination
	c.ShouldBindQuery(&p)

	comments, total, err := h.service.GetPostComments(postID, p.Page, p.Limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, utils.PageResult{
		List:  comments,
		Total: total,
		Page:  p.Page,
		Limit: p.Limit,
	})
}

// ToggleLike 点赞/取消点赞
func (h *MomentHandler) ToggleLike(c *gin.Context) {
	var input LikeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	userID := getUserIdFromContext(c)
	liked, err := h.service.ToggleLike(userID, input.TargetID, input.TargetType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	msg := "unliked"
	if liked {
		msg = "liked"
	}
	response.Success(c, msg)
}

// GetTopics 获取话题列表
// @Summary 获取话题列表
// @Tags Moment
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param keyword query string false "Keyword"
// @Success 200 {object} utils.PageResult
// @Router /moments/topics [get]
func (h *MomentHandler) GetTopics(c *gin.Context) {
	var p utils.Pagination
	c.ShouldBindQuery(&p)
	keyword := c.Query("keyword")

	topics, total, err := h.service.GetTopicList(keyword, p.Page, p.Limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, utils.PageResult{
		List:  topics,
		Total: total,
		Page:  p.Page,
		Limit: p.Limit,
	})
}

// DeleteTopic 删除话题 (管理员)
// @Summary 删除话题
// @Tags Moment
// @Param id path string true "Topic ID"
// @Success 200 {string} string "success"
// @Router /moments/topics/{id} [delete]
func (h *MomentHandler) DeleteTopic(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteTopic(id); err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, "success")
}

func getUserIdFromContext(c *gin.Context) string {
	val, _ := c.Get("userID")
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}
