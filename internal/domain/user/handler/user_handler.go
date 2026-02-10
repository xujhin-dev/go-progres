package handler

import (
	"net/http"
	"strconv"
	"user_crud_jwt/internal/domain/user/service"
	"user_crud_jwt/pkg/response"
	"user_crud_jwt/pkg/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	service service.UserService
}

// NewUserHandler 创建处理器
func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// RegisterInput 注册输入
type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email" binding:"omitempty,email"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// Register 处理注册请求
// @Summary 用户注册
// @Description 注册新用户
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body RegisterInput true "注册信息"
// @Success 200 {string} string "Registration successful"
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	if err := h.service.Register(input.Username, input.Password, input.Email); err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrUserExists, err.Error())
		return
	}

	response.Success(c, "Registration successful")
}

// Login 处理登录请求
func (h *UserHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	token, err := h.service.Login(input.Username, input.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, response.ErrPasswordWrong, err.Error())
		return
	}

	response.Success(c, gin.H{"token": token})
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

	// 将 id 转换为 uint 进行比较
	targetID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid user ID")
		return
	}

	isAdmin := false
	switch v := role.(type) {
	case float64:
		isAdmin = int(v) == 1
	case int:
		isAdmin = v == 1
	}

	if uint(targetID) != currentUserID && !isAdmin {
		response.Error(c, http.StatusForbidden, response.ErrNoPermission, "You can only update your own information")
		return
	}

	user, err := h.service.UpdateUser(id, input.Username, input.Email)
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

	// 将 id 转换为 uint 进行比较
	targetID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid user ID")
		return
	}

	isAdmin := false
	switch v := role.(type) {
	case float64:
		isAdmin = int(v) == 1
	case int:
		isAdmin = v == 1
	}

	if uint(targetID) != currentUserID && !isAdmin {
		response.Error(c, http.StatusForbidden, response.ErrNoPermission, "You can only delete your own account")
		return
	}

	if err := h.service.DeleteUser(id); err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}
	response.Success(c, "User deleted successfully")
}

func getUserIdFromContext(c *gin.Context) uint {
	val, _ := c.Get("userID")
	switch v := val.(type) {
	case uint:
		return v
	case float64:
		return uint(v)
	case int:
		return uint(v)
	default:
		return 0
	}
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Tags User
// @Accept json
// @Produce json
// @Param input body ChangePasswordInput true "密码信息"
// @Success 200 {string} string "Password changed successfully"
// @Router /users/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var input ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	userID := getUserIdFromContext(c)
	if err := h.service.ChangePassword(userID, input.OldPassword, input.NewPassword); err != nil {
		if err.Error() == "invalid old password" {
			response.Error(c, http.StatusBadRequest, response.ErrPasswordWrong, "Invalid old password")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Failed to change password")
		return
	}
	response.Success(c, "Password changed successfully")
}
