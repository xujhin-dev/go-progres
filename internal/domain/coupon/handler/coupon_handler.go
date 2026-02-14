package handler

import (
	"net/http"
	"time"
	"user_crud_jwt/internal/domain/coupon/service"
	"user_crud_jwt/pkg/response"

	"github.com/gin-gonic/gin"
)

type CouponHandler struct {
	service service.CouponService
}

func NewCouponHandler(service service.CouponService) *CouponHandler {
	return &CouponHandler{service: service}
}

type CreateCouponInput struct {
	Name      string    `json:"name" binding:"required"`
	Total     int       `json:"total" binding:"required,min=1"`
	Amount    float64   `json:"amount" binding:"required,min=0.01"`
	StartTime time.Time `json:"startTime" binding:"required"`
	EndTime   time.Time `json:"endTime" binding:"required"`
}

func (h *CouponHandler) CreateCoupon(c *gin.Context) {
	var input CreateCouponInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	coupon, err := h.service.CreateCoupon(input.Name, input.Total, input.Amount, input.StartTime, input.EndTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, coupon)
}

func (h *CouponHandler) ClaimCoupon(c *gin.Context) {
	couponID := c.Param("id")

	// 从 Context 中获取当前登录用户 ID (由 AuthMiddleware 设置)
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, response.ErrTokenInvalid, "User not authenticated")
		return
	}

	// 类型断言为 string
	uid, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Invalid user ID type")
		return
	}

	if err := h.service.ClaimCoupon(uid, couponID); err != nil {
		if err.Error() == "coupon out of stock" || err.Error() == "coupon out of stock (local cache)" {
			response.Fail(c, response.ErrCouponOutOfStock, "Coupon out of stock")
			return
		}
		if err.Error() == "you have already claimed this coupon" {
			response.Fail(c, response.ErrCouponClaimed, "You have already claimed this coupon")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, "Coupon claimed successfully")
}

// SendCouponInput 管理员发券输入
type SendCouponInput struct {
	UserID   string `json:"userId" binding:"required"`
	CouponID string `json:"couponId" binding:"required"`
}

// SendCoupon 管理员给指定用户发券
func (h *CouponHandler) SendCoupon(c *gin.Context) {
	var input SendCouponInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	if err := h.service.SendCouponToUser(input.UserID, input.CouponID); err != nil {
		if err.Error() == "coupon out of stock" || err.Error() == "coupon out of stock (local cache)" {
			response.Fail(c, response.ErrCouponOutOfStock, "Coupon out of stock")
			return
		}
		if err.Error() == "you have already claimed this coupon" {
			response.Fail(c, response.ErrCouponClaimed, "User has already claimed this coupon")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, "Coupon sent to user successfully")
}
