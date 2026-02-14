package security

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"
	"user_crud_jwt/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	config         SecurityConfig
	jwtSecurity    *JWTSecurity
	rateLimiter    RateLimiter
	inputFilter    *InputFilter
	metricsCollector *metrics.MetricsCollector
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableCSRF      bool
	EnableXSS       bool
	EnableCORS      bool
	EnableRateLimit bool
	TrustedProxies  []string
	CORSOrigins     []string
	CORSMethods     []string
	CORSHeaders     []string
	CSRFCookieName  string
	XSSProtection   string
	ContentType     string
	FrameOptions    string
	HSTS            bool
	HSTSMaxAge      int
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableCSRF:      true,
		EnableXSS:       true,
		EnableCORS:      true,
		EnableRateLimit: true,
		TrustedProxies:  []string{"127.0.0.1", "::1"},
		CORSOrigins:     []string{"http://localhost:3000", "http://localhost:8080"},
		CORSMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		CSRFCookieName:  "_csrf_token",
		XSSProtection:   "1; mode=block",
		ContentType:     "nosniff",
		FrameOptions:    "DENY",
		HSTS:           true,
		HSTSMaxAge:      31536000, // 1 year
	}
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(config SecurityConfig, jwtSecurity *JWTSecurity, rateLimiter RateLimiter, inputFilter *InputFilter) *SecurityMiddleware {
	return &SecurityMiddleware{
		config:         config,
		jwtSecurity:    jwtSecurity,
		rateLimiter:    rateLimiter,
		inputFilter:    inputFilter,
		metricsCollector: metrics.GetGlobalCollector(),
	}
}

// Middleware 返回 Gin 中间件
func (sm *SecurityMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 安全头设置
		sm.setSecurityHeaders(c)

		// 2. CORS 处理
		if sm.config.EnableCORS {
			sm.handleCORS(c)
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		}

		// 3. 限流检查
		if sm.config.EnableRateLimit {
			if !sm.checkRateLimit(c) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "rate limit exceeded",
				})
				c.Abort()
				return
			}
		}

		// 4. CSRF 保护
		if sm.config.EnableCSRF && sm.isCSRFRequired(c) {
			if !sm.validateCSRF(c) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "CSRF token validation failed",
				})
				c.Abort()
				return
			}
		}

		// 5. 输入验证
		sm.validateInput(c)

		// 6. 记录安全事件
		sm.logSecurityEvent(c)

		c.Next()
	}
}

// setSecurityHeaders 设置安全头
func (sm *SecurityMiddleware) setSecurityHeaders(c *gin.Context) {
	// XSS 保护
	c.Header("X-XSS-Protection", sm.config.XSSProtection)

	// 内容类型嗅探保护
	c.Header("X-Content-Type-Options", sm.config.ContentType)

	// 点击劫持保护
	c.Header("X-Frame-Options", sm.config.FrameOptions)

	// 引用策略
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

	// 内容安全策略
	csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'"
	c.Header("Content-Security-Policy", csp)

	// HSTS (仅 HTTPS)
	if sm.config.HSTS && c.Request.TLS != nil {
		maxAge := fmt.Sprintf("max-age=%d; includeSubDomains", sm.config.HSTSMaxAge)
		c.Header("Strict-Transport-Security", maxAge)
	}

	// 移除服务器信息
	c.Header("Server", "")
	c.Header("X-Powered-By", "")

	// 权限策略
	c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

	// 缓存控制
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}
}

// handleCORS 处理 CORS
func (sm *SecurityMiddleware) handleCORS(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	
	// 检查是否允许的源
	allowed := false
	for _, allowedOrigin := range sm.config.CORSOrigins {
		if origin == allowedOrigin || allowedOrigin == "*" {
			allowed = true
			break
		}
	}

	if allowed {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", strings.Join(sm.config.CORSMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(sm.config.CORSHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Vary", "Origin")
	}
}

// checkRateLimit 检查限流
func (sm *SecurityMiddleware) checkRateLimit(c *gin.Context) bool {
	// 获取客户端标识
	clientID := sm.getClientID(c)
	
	// 根据路径选择不同的限流策略
	var key string
	var limit Limit
	
	switch {
	case strings.HasPrefix(c.Request.URL.Path, "/auth/"):
		key = fmt.Sprintf("auth:%s", clientID)
		limit = Limit{Rate: 10, Burst: 20, Window: time.Second}
	case strings.HasPrefix(c.Request.URL.Path, "/upload"):
		key = fmt.Sprintf("upload:%s", clientID)
		limit = Limit{Rate: 5, Burst: 10, Window: time.Second}
	default:
		key = fmt.Sprintf("api:%s", clientID)
		limit = Limit{Rate: 100, Burst: 200, Window: time.Second}
	}

	// 设置限流配置
	sm.rateLimiter.SetLimit(context.Background(), key, limit)
	
	// 检查是否允许请求
	allowed, err := sm.rateLimiter.Allow(context.Background(), key)
	if err != nil {
		// 记录错误但允许请求
		sm.metricsCollector.RecordDBError("rate_limit", "check_error")
		return true
	}

	if !allowed {
		// 记录限流事件
		sm.metricsCollector.RecordDBError("rate_limit", "blocked")
	}

	return allowed
}

// getClientID 获取客户端标识
func (sm *SecurityMiddleware) getClientID(c *gin.Context) string {
	// 优先使用用户 ID
	if userID := c.GetString("user_id"); userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}

	// 使用 IP 地址
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// isCSRFRequired 检查是否需要 CSRF 保护
func (sm *SecurityMiddleware) isCSRFRequired(c *gin.Context) bool {
	// 只对状态改变请求进行 CSRF 检查
	method := c.Request.Method
	return method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH"
}

// validateCSRF 验证 CSRF 令牌
func (sm *SecurityMiddleware) validateCSRF(c *gin.Context) bool {
	// 从请求头获取 CSRF 令牌
	headerToken := c.GetHeader("X-CSRF-Token")
	
	// 从 Cookie 获取 CSRF 令牌
	cookieToken, err := c.Cookie(sm.config.CSRFCookieName)
	if err != nil {
		return false
	}

	// 比较令牌
	return subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) == 1
}

// validateInput 验证输入
func (sm *SecurityMiddleware) validateInput(c *gin.Context) {
	// 验证查询参数
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if _, err := sm.inputFilter.FilterInput(value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid query parameter %s: %s", key, err.Error()),
				})
				c.Abort()
				return
			}
		}
	}

	// 验证 JSON 请求体
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") {
			// 这里可以添加 JSON 验证逻辑
			// 由于 Gin 已经解析了 JSON，我们可以在 handler 中进行验证
		}
	}
}

// logSecurityEvent 记录安全事件
func (sm *SecurityMiddleware) logSecurityEvent(c *gin.Context) {
	// 记录可疑的请求
	if sm.isSuspiciousRequest(c) {
		sm.metricsCollector.RecordDBError("security", "suspicious_request")
		// 这里可以添加日志记录或告警
	}
}

// isSuspiciousRequest 检查是否为可疑请求
func (sm *SecurityMiddleware) isSuspiciousRequest(c *gin.Context) bool {
	// 检查 User-Agent
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" || strings.Contains(userAgent, "bot") || strings.Contains(userAgent, "scanner") {
		return true
	}

	// 检查请求路径
	path := c.Request.URL.Path
	suspiciousPaths := []string{"/admin", "/config", "/system", "/debug"}
	for _, suspiciousPath := range suspiciousPaths {
		if strings.Contains(path, suspiciousPath) {
			return true
		}
	}

	// 检查请求参数
	for key := range c.Request.URL.Query() {
		if strings.Contains(key, "sql") || strings.Contains(key, "script") || strings.Contains(key, "alert") {
			return true
		}
	}

	return false
}

// IPWhitelistMiddleware IP 白名单中间件
type IPWhitelistMiddleware struct {
	allowedIPs map[string]bool
}

// NewIPWhitelistMiddleware 创建 IP 白名单中间件
func NewIPWhitelistMiddleware(allowedIPs []string) *IPWhitelistMiddleware {
	ipMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		ipMap[ip] = true
	}
	return &IPWhitelistMiddleware{allowedIPs: ipMap}
}

// Middleware 返回中间件
func (iwm *IPWhitelistMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !iwm.allowedIPs[clientIP] {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "IP address not allowed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware 安全头中间件
type SecurityHeadersMiddleware struct {
	headers map[string]string
}

// NewSecurityHeadersMiddleware 创建安全头中间件
func NewSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
	headers := map[string]string{
		"X-Frame-Options":         "DENY",
		"X-Content-Type-Options":   "nosniff",
		"X-XSS-Protection":         "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
		"Server":                    "",
		"X-Powered-By":             "",
	}

	return &SecurityHeadersMiddleware{headers: headers}
}

// Middleware 返回中间件
func (shm *SecurityHeadersMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		for key, value := range shm.headers {
			c.Header(key, value)
		}
		c.Next()
	}
}

// AddHeader 添加安全头
func (shm *SecurityHeadersMiddleware) AddHeader(key, value string) {
	shm.headers[key] = value
}

// RequestSizeMiddleware 请求大小限制中间件
type RequestSizeMiddleware struct {
	maxSize int64
}

// NewRequestSizeMiddleware 创建请求大小限制中间件
func NewRequestSizeMiddleware(maxSize int64) *RequestSizeMiddleware {
	return &RequestSizeMiddleware{maxSize: maxSize}
}

// Middleware 返回中间件
func (rsm *RequestSizeMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查 Content-Length
		if contentLength := c.Request.ContentLength; contentLength > rsm.maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("request too large, maximum size is %d bytes", rsm.maxSize),
			})
			c.Abort()
			return
		}

		// 限制请求体大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, rsm.maxSize)
		c.Next()
	}
}

// TimeoutMiddleware 超时中间件
type TimeoutMiddleware struct {
	timeout time.Duration
}

// NewTimeoutMiddleware 创建超时中间件
func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{timeout: timeout}
}

// Middleware 返回中间件
func (tm *TimeoutMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), tm.timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AuditMiddleware 审计中间件
type AuditMiddleware struct {
	logger Logger
}

// Logger 日志接口
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// NewAuditMiddleware 创建审计中间件
func NewAuditMiddleware(logger Logger) *AuditMiddleware {
	return &AuditMiddleware{logger: logger}
}

// Middleware 返回中间件
func (am *AuditMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// 记录请求开始
		am.logRequest(c, "request_start")
		
		c.Next()
		
		// 记录请求结束
		duration := time.Since(start)
		am.logRequest(c, "request_end", "duration", duration.Milliseconds())
	}
}

// logRequest 记录请求
func (am *AuditMiddleware) logRequest(c *gin.Context, event string, fields ...interface{}) {
	data := map[string]interface{}{
		"event":     event,
		"method":    c.Request.Method,
		"path":      c.Request.URL.Path,
		"query":      c.Request.URL.RawQuery,
		"user_agent": c.GetHeader("User-Agent"),
		"ip":        c.ClientIP(),
		"status":    c.Writer.Status(),
	}
	
	// 添加额外字段
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			data[fields[i].(string)] = fields[i+1]
		}
	}
	
	// 根据状态选择日志级别
	switch c.Writer.Status() {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		am.logger.Info("audit", data)
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		am.logger.Warn("audit", data)
	default:
		if c.Writer.Status() >= 500 {
			am.logger.Error("audit", data)
		} else {
			am.logger.Info("audit", data)
		}
	}
}
