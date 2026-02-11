package middleware

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// PerformanceMiddleware 性能优化中间件
type PerformanceMiddleware struct {
	metricsCollector *metrics.MetricsCollector
	rateLimiter      *RateLimiter
	circuitBreaker   *CircuitBreaker
	enableGzip       bool
	enableMetrics    bool
}

// NewPerformanceMiddleware 创建性能中间件
func NewPerformanceMiddleware(enableMetrics bool) *PerformanceMiddleware {
	return &PerformanceMiddleware{
		metricsCollector: metrics.GetGlobalCollector(),
		rateLimiter:      NewRateLimiter(1000, 2000), // 1000 req/s, burst 2000
		circuitBreaker:   NewCircuitBreaker(10, time.Minute*5),
		enableGzip:       true,
		enableMetrics:    enableMetrics,
	}
}

// Middleware 返回 Gin 中间件
func (pm *PerformanceMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 限流检查
		if !pm.rateLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		// 熔断器检查
		if err := pm.circuitBreaker.Call(func() error {
			pm.processRequest(c)
			return nil
		}); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "service unavailable",
			})
			c.Abort()
			return
		}

		// 记录指标
		if pm.enableMetrics {
			duration := time.Since(start)
			method := c.Request.Method
			endpoint := c.FullPath()
			status := c.Writer.Status()

			requestSize := getContentLength(c.Request)
			responseSize := c.Writer.Size()

			pm.metricsCollector.RecordHTTPRequest(
				method,
				endpoint,
				strconv.Itoa(status),
				duration,
				requestSize,
				responseSize,
			)
		}
	}
}

// processRequest 处理请求
func (pm *PerformanceMiddleware) processRequest(c *gin.Context) {
	// Gzip 压缩
	if pm.enableGzip && shouldCompress(c.Request) {
		pm.gzipHandler(c)
	} else {
		c.Next()
	}
}

// gzipHandler Gzip 处理器
func (pm *PerformanceMiddleware) gzipHandler(c *gin.Context) {
	c.Header("Content-Encoding", "gzip")
	c.Header("Vary", "Accept-Encoding")

	gz := gzip.NewWriter(c.Writer)
	defer gz.Close()

	c.Writer = &gzipWriter{
		ResponseWriter: c.Writer,
		Writer:         gz,
	}

	c.Next()
}

// gzipWriter Gzip 写入器
type gzipWriter struct {
	gin.ResponseWriter
	Writer *gzip.Writer
}

func (gw *gzipWriter) Write(data []byte) (int, error) {
	return gw.Writer.Write(data)
}

// shouldCompress 检查是否应该压缩
func shouldCompress(req *http.Request) bool {
	if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}

	// 不压缩已经压缩的内容
	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "gzip") ||
		strings.Contains(contentType, "deflate") ||
		strings.Contains(contentType, "br") {
		return false
	}

	return true
}

// getContentLength 获取内容长度
func getContentLength(req *http.Request) int {
	if req.ContentLength > 0 {
		return int(req.ContentLength)
	}

	// 如果 Content-Length 为 -1，尝试读取 body
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		return len(body)
	}

	return 0
}

// RateLimiter 限流器
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(refillRate, maxTokens int) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	rl.lastRefill = now

	// 补充令牌
	tokensToAdd := int(elapsed.Seconds()) * rl.refillRate
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
	}

	// 检查是否有令牌
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        CircuitState
	mu           sync.RWMutex
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// Call 执行函数调用
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 检查状态
	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return ErrCircuitBreakerOpen
		}
	}

	// 执行函数
	err := fn()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}

		return err
	}

	// 成功时重置
	cb.failures = 0
	cb.state = StateClosed

	return nil
}

// ErrCircuitBreakerOpen 熔断器开启错误
var ErrCircuitBreakerOpen = &circuitBreakerError{"circuit breaker is open"}

type circuitBreakerError struct {
	message string
}

func (e *circuitBreakerError) Error() string {
	return e.message
}

// SystemMetricsMiddleware 系统指标中间件
func SystemMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 更新系统指标
		collector := metrics.GetGlobalCollector()

		// 更新 goroutine 数量
		collector.UpdateActiveGoroutines(runtime.NumGoroutine())

		// 更新内存使用
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		collector.UpdateMemoryUsage(int(m.Alloc))

		c.Next()
	}
}

// RequestIDMiddleware 请求 ID 中间件（优化版）
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已有请求 ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成更高效的请求 ID
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	// 使用时间戳和随机数生成更高效的 ID
	timestamp := time.Now().UnixNano()
	// 简单的随机数生成
	randNum := timestamp % 1000000
	return strconv.FormatInt(timestamp, 36) + strconv.FormatInt(randNum, 36)
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			c.Next()
			close(finished)
		}()

		select {
		case <-finished:
			// 正常完成
		case <-ctx.Done():
			// 超时
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "request timeout",
			})
			c.Abort()
		}
	}
}

// RecoveryMiddleware 恢复中间件（优化版）
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// 记录错误指标
		collector := metrics.GetGlobalCollector()
		collector.RecordDBError("panic", "recovery")

		// 返回错误响应
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	})
}

// CORS Middleware (优化版)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许的源
		if isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isAllowedOrigin 检查是否允许的源
func isAllowedOrigin(origin string) bool {
	// 这里可以实现更复杂的源检查逻辑
	// 例如从配置文件读取允许的域名列表

	allowedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:8080",
		"https://yourdomain.com",
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}

	return false
}

// SecurityHeadersMiddleware 安全头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}
