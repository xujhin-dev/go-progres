package middleware

import (
	"net/http"
	"sync"

	"user_crud_jwt/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter 存储每个IP的限流器
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter 创建一个新的IP限流器
// r: 每秒允许的请求数 (QPS)
// b: 桶的大小 (Burst)
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	// 启动清理协程，定期清理过期的IP（这里简化处理，实际生产可以使用LRU缓存或Redis）
	return i
}

// GetLimiter 获取指定IP的限流器
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

// GlobalRateLimiter 全局限流器实例
// 默认限制：每秒 10000 个请求，突发 20000 个 (为了演示高并发，设置得比较大)
var limiter = NewIPRateLimiter(10000, 20000)

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.GetLimiter(ip)
		if !l.Allow() {
			response.Error(c, http.StatusTooManyRequests, response.ErrTooManyRequests, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
