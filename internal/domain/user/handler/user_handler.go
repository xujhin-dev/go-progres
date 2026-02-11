package handler

import (
	"net/http"
	"user_crud_jwt/internal/domain/user/service"
	"user_crud_jwt/pkg/response"
	"user_crud_jwt/pkg/utils"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

type LoginInput struct {
	Mobile string `json:"mobile" binding:"required,len=11"`
	Code   string `json:"code" binding:"required,len=6"`
}

type OTPInput struct {
	Mobile string `json:"mobile" binding:"required,len=11"`
}

type UpdateUserInput struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// LoginOrRegister 登录/注册
// @Summary 手机号验证码登录
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body LoginInput true "登录信息"
// @Success 200 {object} response.Response{data=string} "Token"
// @Router /auth/login [post]
func (h *UserHandler) LoginOrRegister(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	token, err := h.service.LoginOrRegister(input.Mobile, input.Code)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, response.ErrAuthFailed, err.Error())
		return
	}
	response.Success(c, token)
}

// SendOTP 发送验证码
// @Summary 发送验证码
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body OTPInput true "手机号"
// @Success 200 {string} string "success"
// @Router /auth/otp [post]
func (h *UserHandler) SendOTP(c *gin.Context) {
	var input OTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	if err := h.service.SendOTP(input.Mobile); err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, "success")
}

// GetUsers 获取所有用户
// @Summary 获取用户列表
// @Description 分页获取用户列表
// @Tags User
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} utils.PageResult
// @Router /users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	var pagination utils.Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	users, total, err := h.service.GetUsers(pagination.Page, pagination.Limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Failed to fetch users")
		return
	}

	result := utils.PageResult{
		List:  users,
		Total: total,
		Page:  pagination.Page,
		Limit: pagination.Limit,
	}
	response.Success(c, result)
}

// GetUser 获取单个用户
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := h.service.GetUser(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, response.ErrUserNotFound, "User not found")
		return
	}
	response.Success(c, user)
}

// UpdateUser 更新用户
// @Summary 更新用户信息
// @Description 更新用户名或邮箱
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param input body UpdateUserInput true "更新信息"
// @Success 200 {object} model.User
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	// 权限校验：只能修改自己的信息，或者管理员可以修改任何人
	currentUserID := getUserIdFromContext(c)
	role, _ := c.Get("role")

	isAdmin := false
	switch v := role.(type) {
	case float64:
		isAdmin = int(v) == 1
	case int:
		isAdmin = v == 1
	}

	if id != currentUserID && !isAdmin {
		response.Error(c, http.StatusForbidden, response.ErrNoPermission, "You can only update your own information")
		return
	}

	user, err := h.service.UpdateUser(id, input.Nickname, input.AvatarURL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Failed to update user")
		return
	}
	response.Success(c, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// 权限校验：只能删除自己的账号，或者管理员可以删除任何人
	currentUserID := getUserIdFromContext(c)
	role, _ := c.Get("role")

	isAdmin := false
	switch v := role.(type) {
	case float64:
		isAdmin = int(v) == 1
	case int:
		isAdmin = v == 1
	}

	if id != currentUserID && !isAdmin {
		response.Error(c, http.StatusForbidden, response.ErrNoPermission, "You can only delete your own account")
		return
	}

	if err := h.service.DeleteUser(id); err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, "User deleted successfully")
}

func getUserIdFromContext(c *gin.Context) string {
	val, _ := c.Get("userID")
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}


