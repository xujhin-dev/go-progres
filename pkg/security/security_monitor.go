package security

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/cache"
	"user_crud_jwt/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// SecurityEvent 安全事件
type SecurityEvent struct {
	ID        string                 `json:"id"`
	Type      SecurityEventType      `json:"type"`
	Level     SecurityEventLevel     `json:"level"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	UserID    string                 `json:"user_id,omitempty"`
	IP        string                 `json:"ip"`
	UserAgent string                 `json:"user_agent"`
	Path      string                 `json:"path"`
	Method    string                 `json:"method"`
	Status    int                    `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SecurityEventType 安全事件类型
type SecurityEventType string

const (
	EventLogin            SecurityEventType = "login"
	EventLogout           SecurityEventType = "logout"
	EventTokenExpired     SecurityEventType = "token_expired"
	EventTokenRevoked     SecurityEventType = "token_revoked"
	EventRateLimit        SecurityEventType = "rate_limit"
	EventSuspicious       SecurityEventType = "suspicious"
	EventXSS              SecurityEventType = "xss"
	EventSQLInjection     SecurityEventType = "sql_injection"
	EventCSRF             SecurityEventType = "csrf"
	EventUnauthorized     SecurityEventType = "unauthorized"
	EventForbidden        SecurityEventType = "forbidden"
	EventInputValidation  SecurityEventType = "input_validation"
	EventPermissionDenied SecurityEventType = "permission_denied"
)

// SecurityEventLevel 安全事件级别
type SecurityEventLevel string

const (
	LevelInfo     SecurityEventLevel = "info"
	LevelWarning  SecurityEventLevel = "warning"
	LevelError    SecurityEventLevel = "error"
	LevelCritical SecurityEventLevel = "critical"
)

// SecurityMonitor 安全监控器
type SecurityMonitor struct {
	cache            cache.CacheService
	metricsCollector *metrics.MetricsCollector
	events           []SecurityEvent
	mu               sync.RWMutex
	alertThresholds  map[SecurityEventType]int
	alertHandlers    []AlertHandler
	logger           SecurityLogger
}

// AlertHandler 告警处理器接口
type AlertHandler interface {
	Handle(event SecurityEvent) error
}

// NewSecurityMonitor 创建安全监控器
func NewSecurityMonitor(cache cache.CacheService, metricsCollector *metrics.MetricsCollector, logger SecurityLogger) *SecurityMonitor {
	return &SecurityMonitor{
		cache:            cache,
		metricsCollector: metricsCollector,
		events:           make([]SecurityEvent, 0),
		alertThresholds: map[SecurityEventType]int{
			EventRateLimit:    10, // 10次/分钟
			EventSuspicious:   5,  // 5次/分钟
			EventUnauthorized: 20, // 20次/分钟
			EventForbidden:    10, // 10次/分钟
		},
		alertHandlers: make([]AlertHandler, 0),
		logger:        logger,
	}
}

// RecordEvent 记录安全事件
func (sm *SecurityMonitor) RecordEvent(event SecurityEvent) {
	// 设置时间戳
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 生成 ID
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// 存储事件
	sm.mu.Lock()
	sm.events = append(sm.events, event)

	// 保持最近1000个事件
	if len(sm.events) > 1000 {
		sm.events = sm.events[len(sm.events)-1000:]
	}
	sm.mu.Unlock()

	// 记录到缓存
	sm.cacheEvent(event)

	// 记录指标
	sm.recordMetrics(event)

	// 记录日志
	sm.logEvent(event)

	// 检查告警
	sm.checkAlerts(event)
}

// cacheEvent 缓存事件
func (sm *SecurityMonitor) cacheEvent(event SecurityEvent) {
	cacheKey := fmt.Sprintf("security_event:%s", event.ID)
	sm.cache.Set(context.Background(), cacheKey, event, time.Hour*24)
}

// recordMetrics 记录指标
func (sm *SecurityMonitor) recordMetrics(event SecurityEvent) {
	// 记录安全事件计数
	sm.metricsCollector.RecordDBError("security_event", string(event.Type))

	// 根据事件级别记录不同的指标
	switch event.Level {
	case LevelCritical:
		sm.metricsCollector.RecordDBError("security_critical", string(event.Type))
	case LevelError:
		sm.metricsCollector.RecordDBError("security_error", string(event.Type))
	case LevelWarning:
		sm.metricsCollector.RecordDBError("security_warning", string(event.Type))
	}
}

// logEvent 记录日志
func (sm *SecurityMonitor) logEvent(event SecurityEvent) {
	logData := map[string]interface{}{
		"event_id":  event.ID,
		"type":      event.Type,
		"level":     event.Level,
		"timestamp": event.Timestamp,
		"source":    event.Source,
		"ip":        event.IP,
		"path":      event.Path,
		"method":    event.Method,
		"status":    event.Status,
		"message":   event.Message,
	}

	if event.UserID != "" {
		logData["user_id"] = event.UserID
	}

	if event.UserAgent != "" {
		logData["user_agent"] = event.UserAgent
	}

	if len(event.Details) > 0 {
		logData["details"] = event.Details
	}

	switch event.Level {
	case LevelCritical:
		sm.logger.Critical("Security event", logData)
	case LevelError:
		sm.logger.Error("Security event", logData)
	case LevelWarning:
		sm.logger.Warn("Security event", logData)
	default:
		sm.logger.Info("Security event", logData)
	}
}

// checkAlerts 检查告警
func (sm *SecurityMonitor) checkAlerts(event SecurityEvent) {
	// 检查事件频率阈值
	if threshold, exists := sm.alertThresholds[event.Type]; exists {
		count := sm.getEventCount(event.Type, time.Minute)
		if count >= threshold {
			sm.triggerAlert(event, fmt.Sprintf("Event threshold exceeded: %d events in 1 minute", count))
		}
	}

	// 检查关键事件
	if event.Level == LevelCritical {
		sm.triggerAlert(event, "Critical security event detected")
	}
}

// getEventCount 获取事件计数
func (sm *SecurityMonitor) getEventCount(eventType SecurityEventType, duration time.Duration) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	since := time.Now().Add(-duration)

	for _, event := range sm.events {
		if event.Type == eventType && event.Timestamp.After(since) {
			count++
		}
	}

	return count
}

// triggerAlert 触发告警
func (sm *SecurityMonitor) triggerAlert(event SecurityEvent, reason string) {
	alert := Alert{
		ID:        generateAlertID(),
		Event:     event,
		Reason:    reason,
		Timestamp: time.Now(),
		Handled:   false,
	}

	for _, handler := range sm.alertHandlers {
		if err := handler.Handle(alert.Event); err != nil {
			sm.logger.Error("Alert handler failed", "error", err)
		}
	}
}

// GetEvents 获取事件列表
func (sm *SecurityMonitor) GetEvents(eventType SecurityEventType, limit int) []SecurityEvent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var events []SecurityEvent
	for _, event := range sm.events {
		if eventType == "" || event.Type == eventType {
			events = append(events, event)
			if len(events) >= limit {
				break
			}
		}
	}

	return events
}

// GetEventStats 获取事件统计
func (sm *SecurityMonitor) GetEventStats() map[string]int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]int)
	for _, event := range sm.events {
		stats[string(event.Type)]++
	}

	return stats
}

// AddAlertHandler 添加告警处理器
func (sm *SecurityMonitor) AddAlertHandler(handler AlertHandler) {
	sm.alertHandlers = append(sm.alertHandlers, handler)
}

// SetAlertThreshold 设置告警阈值
func (sm *SecurityMonitor) SetAlertThreshold(eventType SecurityEventType, threshold int) {
	sm.alertThresholds[eventType] = threshold
}

// Alert 告警信息
type Alert struct {
	ID        string        `json:"id"`
	Event     SecurityEvent `json:"event"`
	Reason    string        `json:"reason"`
	Timestamp time.Time     `json:"timestamp"`
	Handled   bool          `json:"handled"`
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// generateAlertID 生成告警ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// SecurityMiddleware 安全监控中间件
type SecurityMonitoringMiddleware struct {
	monitor *SecurityMonitor
}

// NewSecurityMonitoringMiddleware 创建安全监控中间件
func NewSecurityMonitoringMiddleware(monitor *SecurityMonitor) *SecurityMonitoringMiddleware {
	return &SecurityMonitoringMiddleware{monitor: monitor}
}

// Middleware 返回中间件
func (smm *SecurityMonitoringMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在请求处理前记录
		c.Set("security_start_time", time.Now())

		// 处理请求
		c.Next()

		// 在请求处理后检查安全事件
		sm.checkSecurityEvents(c)
	}
}

// checkSecurityEvents 检查安全事件
func (smm *SecurityMonitoringMiddleware) checkSecurityEvents(c *gin.Context) {
	startTime, exists := c.Get("security_start_time")
	if !exists {
		return
	}

	duration := time.Since(startTime.(time.Time))
	status := c.Writer.Status()
	path := c.Request.URL.Path
	method := c.Request.Method
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 检查不同的安全事件
	switch {
	case status == 401:
		smm.monitor.RecordEvent(SecurityEvent{
			Type:      EventUnauthorized,
			Level:     LevelWarning,
			Source:    "api",
			UserID:    toString(userID),
			IP:        ip,
			UserAgent: userAgent,
			Path:      path,
			Method:    method,
			Status:    status,
			Message:   "Unauthorized access attempt",
			Details: map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			},
		})

	case status == 403:
		smm.monitor.RecordEvent(SecurityEvent{
			Type:      EventForbidden,
			Level:     LevelWarning,
			Source:    "api",
			UserID:    toString(userID),
			IP:        ip,
			UserAgent: userAgent,
			Path:      path,
			Method:    method,
			Status:    status,
			Message:   "Forbidden access attempt",
			Details: map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			},
		})

	case status >= 500:
		smm.monitor.RecordEvent(SecurityEvent{
			Type:      "server_error",
			Level:     LevelError,
			Source:    "api",
			UserID:    toString(userID),
			IP:        ip,
			UserAgent: userAgent,
			Path:      path,
			Method:    method,
			Status:    status,
			Message:   "Server error occurred",
			Details: map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			},
		})

	case duration > time.Second*5:
		smm.monitor.RecordEvent(SecurityEvent{
			Type:      "slow_request",
			Level:     LevelWarning,
			Source:    "api",
			UserID:    toString(userID),
			IP:        ip,
			UserAgent: userAgent,
			Path:      path,
			Method:    method,
			Status:    status,
			Message:   "Slow request detected",
			Details: map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			},
		})
	}

	// 检查可疑请求
	if smm.isSuspiciousRequest(c) {
		smm.monitor.RecordEvent(SecurityEvent{
			Type:      EventSuspicious,
			Level:     LevelWarning,
			Source:    "api",
			UserID:    toString(userID),
			IP:        ip,
			UserAgent: userAgent,
			Path:      path,
			Method:    method,
			Status:    status,
			Message:   "Suspicious request detected",
			Details: map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			},
		})
	}
}

// isSuspiciousRequest 检查是否为可疑请求
func (smm *SecurityMonitoringMiddleware) isSuspiciousRequest(c *gin.Context) bool {
	// 检查 User-Agent
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" || strings.Contains(userAgent, "bot") || strings.Contains(userAgent, "scanner") {
		return true
	}

	// 检查请求路径
	path := c.Request.URL.Path
	suspiciousPaths := []string{"/admin", "/config", "/system", "/debug", "/env", "/proc"}
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

// toString 安全转换 interface{} 到 string
func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

// EmailAlertHandler 邮件告警处理器
type EmailAlertHandler struct {
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
	toEmails     []string
}

// NewEmailAlertHandler 创建邮件告警处理器
func NewEmailAlertHandler(smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail string, toEmails []string) *EmailAlertHandler {
	return &EmailAlertHandler{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
		toEmails:     toEmails,
	}
}

// Handle 处理告警
func (eah *EmailAlertHandler) Handle(event SecurityEvent) error {
	// 这里可以实现邮件发送逻辑
	// 为了简化，这里只是记录日志
	log.Printf("Security Alert: %s - %s", event.Type, event.Message)
	return nil
}

// SlackAlertHandler Slack 告警处理器
type SlackAlertHandler struct {
	webhookURL string
	channel    string
}

// NewSlackAlertHandler 创建 Slack 告警处理器
func NewSlackAlertHandler(webhookURL, channel string) *SlackAlertHandler {
	return &SlackAlertHandler{
		webhookURL: webhookURL,
		channel:    channel,
	}
}

// Handle 处理告警
func (sah *SlackAlertHandler) Handle(event SecurityEvent) error {
	// 这里可以实现 Slack Webhook 调用
	// 为了简化，这里只是记录日志
	log.Printf("Slack Alert: %s - %s", event.Type, event.Message)
	return nil
}

// SecurityMetrics 安全指标
type SecurityMetrics struct {
	TotalEvents    int64            `json:"total_events"`
	CriticalEvents int64            `json:"critical_events"`
	ErrorEvents    int64            `json:"error_events"`
	WarningEvents  int64            `json:"warning_events"`
	InfoEvents     int64            `json:"info_events"`
	LastEventTime  time.Time        `json:"last_event_time"`
	EventsByType   map[string]int64 `json:"events_by_type"`
	EventsByHour   map[string]int64 `json:"events_by_hour"`
}

// GetMetrics 获取安全指标
func (sm *SecurityMonitor) GetMetrics() SecurityMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics := SecurityMetrics{
		TotalEvents:  int64(len(sm.events)),
		EventsByType: make(map[string]int64),
		EventsByHour: make(map[string]int64),
	}

	if len(sm.events) > 0 {
		metrics.LastEventTime = sm.events[len(sm.events)-1].Timestamp
	}

	for _, event := range sm.events {
		metrics.EventsByType[string(event.Type)]++

		hour := event.Timestamp.Format("2006-01-02-15")
		metrics.EventsByHour[hour]++

		switch event.Level {
		case LevelCritical:
			metrics.CriticalEvents++
		case LevelError:
			metrics.ErrorEvents++
		case LevelWarning:
			metrics.WarningEvents++
		case LevelInfo:
			metrics.InfoEvents++
		}
	}

	return metrics
}

// GenerateReport 生成安全报告
func (sm *SecurityMonitor) GenerateReport(duration time.Duration) SecurityReport {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	report := SecurityReport{
		Period:    duration,
		StartTime: time.Now().Add(-duration),
		EndTime:   time.Now(),
		Metrics:   sm.GetMetrics(),
		Events:    sm.getEventsInPeriod(duration),
		TopEvents: sm.getTopEvents(10),
	}

	return report
}

// SecurityReport 安全报告
type SecurityReport struct {
	Period    time.Duration   `json:"period"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	Metrics   SecurityMetrics `json:"metrics"`
	Events    []SecurityEvent `json:"events"`
	TopEvents []EventCount    `json:"top_events"`
}

// EventCount 事件计数
type EventCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// getEventsInPeriod 获取时间段内的事件
func (sm *SecurityMonitor) getEventsInPeriod(duration time.Duration) []SecurityEvent {
	since := time.Now().Add(-duration)
	var events []SecurityEvent

	for _, event := range sm.events {
		if event.Timestamp.After(since) {
			events = append(events, event)
		}
	}

	return events
}

// getTopEvents 获取最频繁的事件
func (sm *SecurityMonitor) getTopEvents(limit int) []EventCount {
	eventCounts := make(map[string]int)

	for _, event := range sm.events {
		eventCounts[string(event.Type)]++
	}

	var topEvents []EventCount
	for eventType, count := range eventCounts {
		topEvents = append(topEvents, EventCount{
			Type:  eventType,
			Count: count,
		})
	}

	// 简单排序（实际项目中应该使用更高效的排序）
	for i := 0; i < len(topEvents)-1; i++ {
		for j := i + 1; j < len(topEvents); j++ {
			if topEvents[i].Count < topEvents[j].Count {
				topEvents[i], topEvents[j] = topEvents[j], topEvents[i]
			}
		}
	}

	if len(topEvents) > limit {
		topEvents = topEvents[:limit]
	}

	return topEvents
}
