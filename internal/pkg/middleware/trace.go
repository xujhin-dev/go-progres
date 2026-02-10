package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceMiddleware 添加请求追踪ID
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取 TraceID，如果没有则生成新的
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 设置到 context 和响应头
		c.Set("traceID", traceID)
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
