package handler

import (
	"net/http"
	"user_crud_jwt/internal/domain/payment/service"
	"user_crud_jwt/pkg/response"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(s service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: s}
}

type CreateOrderInput struct {
	Amount  float64 `json:"amount" binding:"required,gt=0"`
	Channel string  `json:"channel" binding:"required,oneof=alipay wechat"`
	Subject string  `json:"subject" binding:"required"`
}

// CreateOrder 创建订单
// @Summary 创建订单
// @Tags Payment
// @Accept json
// @Produce json
// @Param input body CreateOrderInput true "Order Info"
// @Success 200 {object} response.Response{data=string} "Pay Param"
// @Router /payment/order [post]
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	var input CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, err.Error())
		return
	}

	userID := getUserIdFromContext(c)
	order, payParam, err := h.service.CreateOrder(userID, input.Amount, input.Channel, input.Subject)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, err.Error())
		return
	}

	response.Success(c, gin.H{
		"order_no":  order.OrderNo,
		"pay_param": payParam,
	})
}

// AlipayNotify 支付宝回调
// @Summary 支付宝回调
// @Tags Payment
// @Router /payment/notify/alipay [post]
func (h *PaymentHandler) AlipayNotify(c *gin.Context) {
	// 支付宝回调是 POST Form 格式
	c.Request.ParseForm()
	err := h.service.HandleNotify("alipay", c.Request.Form)
	if err != nil {
		c.String(http.StatusOK, "fail") // 告诉支付宝处理失败，它会重试
		return
	}
	c.String(http.StatusOK, "success")
}

// WechatNotify 微信支付回调
// @Summary 微信支付回调
// @Tags Payment
// @Router /payment/notify/wechat [post]
func (h *PaymentHandler) WechatNotify(c *gin.Context) {
	// 微信支付回调是 JSON 格式，且需要从 Header 获取签名信息
	// 传递 *http.Request 给 Strategy 处理
	err := h.service.HandleNotify("wechat", c.Request)
	if err != nil {
		// 返回 4xx/5xx 表示失败
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}
	// 返回 2xx 表示成功
	c.Status(http.StatusOK)
}

func getUserIdFromContext(c *gin.Context) string {
	val, _ := c.Get("userID")
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}
