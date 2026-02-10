package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`    // 业务码
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 数据
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpCode int, errCode int, msg string) {
	c.JSON(httpCode, Response{
		Code:    errCode,
		Message: msg,
		Data:    nil,
	})
}

// Fail 业务失败响应 (HTTP 200, 业务码非 0)
func Fail(c *gin.Context, errCode int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code:    errCode,
		Message: msg,
		Data:    nil,
	})
}
