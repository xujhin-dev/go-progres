package handler

import (
	"net/http"
	"strconv"
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
	couponID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid coupon ID")
		return
	}

	// 从 Context 中获取当前登录用户 ID (由 AuthMiddleware 设置)
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, response.ErrTokenInvalid, "User not authenticated")
		return
	}

	// 类型断言
	uid, ok := userID.(float64)
	if !ok {
		// 尝试转 int (取决于 JWT 解析后的类型)
		if uidInt, ok := userID.(int); ok {
			uid = float64(uidInt)
		} else {
			response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Invalid user ID type")
			return
		}
	}

	if err := h.service.ClaimCoupon(uint(uid), uint(couponID)); err != nil {
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
	UserID   uint `json:"userId" binding:"required"`
	CouponID uint `json:"couponId" binding:"required"`
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
