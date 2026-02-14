package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"
)

// CacheMonitor 缓存监控器
type CacheMonitor struct {
	cache            CacheService
	metricsCollector *metrics.MetricsCollector
	config           *MonitorConfig
	stats            *CacheStats
	alerter          *CacheAlerter
	reporter         *CacheReporter
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	MonitorInterval time.Duration    `json:"monitor_interval"`
	EnableAlerts    bool             `json:"enable_alerts"`
	EnableReporting bool             `json:"enable_reporting"`
	ReportInterval  time.Duration    `json:"report_interval"`
	MaxHistorySize  int              `json:"max_history_size"`
	EnableMetrics   bool             `json:"enable_metrics"`
	AlertThresholds *AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds 告警阈值
type AlertThresholds struct {
	HitRateMin      float64       `json:"hit_rate_min"`
	MissRateMax     float64       `json:"miss_rate_max"`
	ResponseTimeMax time.Duration `json:"response_time_max"`
	ErrorRateMax    float64       `json:"error_rate_max"`
	MemoryUsageMax  int64         `json:"memory_usage_max"`
	ConnectionsMax  int           `json:"connections_max"`
}

// CacheStats 缓存统计
type CacheStats struct {
	TotalRequests     int64           `json:"total_requests"`
	HitRequests       int64           `json:"hit_requests"`
	MissRequests      int64           `json:"miss_requests"`
	ErrorRequests     int64           `json:"error_requests"`
	HitRate           float64         `json:"hit_rate"`
	MissRate          float64         `json:"miss_rate"`
	ErrorRate         float64         `json:"error_rate"`
	AvgResponseTime   time.Duration   `json:"avg_response_time"`
	P95ResponseTime   time.Duration   `json:"p95_response_time"`
	P99ResponseTime   time.Duration   `json:"p99_response_time"`
	MemoryUsage       int64           `json:"memory_usage"`
	ActiveConnections int             `json:"active_connections"`
	LastReset         time.Time       `json:"last_reset"`
	History           []CacheSnapshot `json:"history"`
}

// CacheSnapshot 缓存快照
type CacheSnapshot struct {
	Timestamp         time.Time     `json:"timestamp"`
	TotalRequests     int64         `json:"total_requests"`
	HitRequests       int64         `json:"hit_requests"`
	MissRequests      int64         `json:"miss_requests"`
	ErrorRequests     int64         `json:"error_requests"`
	HitRate           float64       `json:"hit_rate"`
	MissRate          float64       `json:"miss_rate"`
	ErrorRate         float64       `json:"error_rate"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	MemoryUsage       int64         `json:"memory_usage"`
	ActiveConnections int           `json:"active_connections"`
}

// CacheAlerter 缓存告警器
type CacheAlerter struct {
	config *MonitorConfig
	alerts []CacheAlert
	mu     sync.RWMutex
}

// CacheAlert 缓存告警
type CacheAlert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Resolved  bool      `json:"resolved"`
}

// CacheReporter 缓存报告器
type CacheReporter struct {
	config *MonitorConfig
}

// CacheReport 缓存报告
type CacheReport struct {
	Period          time.Duration `json:"period"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Summary         CacheSummary  `json:"summary"`
	Details         ReportDetails `json:"details"`
	Alerts          []CacheAlert  `json:"alerts"`
	Recommendations []string      `json:"recommendations"`
}

// CacheSummary 缓存摘要
type CacheSummary struct {
	TotalRequests    int64         `json:"total_requests"`
	HitRate          float64       `json:"hit_rate"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	ErrorRate        float64       `json:"error_rate"`
	MemoryUsage      int64         `json:"memory_usage"`
	PerformanceScore float64       `json:"performance_score"`
}

// ReportDetails 报告详情
type ReportDetails struct {
	TopKeys           []KeyStats   `json:"top_keys"`
	ResponseTimeStats []TimeStats  `json:"response_time_stats"`
	ErrorStats        []ErrorStats `json:"error_stats"`
	Trends            []TrendData  `json:"trends"`
}

// KeyStats 键统计
type KeyStats struct {
	Key         string    `json:"key"`
	AccessCount int64     `json:"access_count"`
	HitCount    int64     `json:"hit_count"`
	HitRate     float64   `json:"hit_rate"`
	LastAccess  time.Time `json:"last_access"`
}

// TimeStats 时间统计
type TimeStats struct {
	Percentile string        `json:"percentile"`
	Value      time.Duration `json:"value"`
}

// ErrorStats 错误统计
type ErrorStats struct {
	ErrorType string    `json:"error_type"`
	Count     int64     `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
}

// TrendData 趋势数据
type TrendData struct {
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
}

// NewCacheMonitor 创建缓存监控器
func NewCacheMonitor(cache CacheService, metricsCollector *metrics.MetricsCollector, config *MonitorConfig) *CacheMonitor {
	return &CacheMonitor{
		cache:            cache,
		metricsCollector: metricsCollector,
		config:           config,
		stats:            &CacheStats{History: make([]CacheSnapshot, 0)},
		alerter:          NewCacheAlerter(config),
		reporter:         NewCacheReporter(config),
	}
}

// NewCacheAlerter 创建缓存告警器
func NewCacheAlerter(config *MonitorConfig) *CacheAlerter {
	return &CacheAlerter{
		config: config,
		alerts: make([]CacheAlert, 0),
	}
}

// NewCacheReporter 创建缓存报告器
func NewCacheReporter(config *MonitorConfig) *CacheReporter {
	return &CacheReporter{
		config: config,
	}
}

// Start 开始监控
func (cm *CacheMonitor) Start() {
	ticker := time.NewTicker(cm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.collectStats()
		}
	}
}

// collectStats 收集统计信息
func (cm *CacheMonitor) collectStats() {
	start := time.Now()

	// 创建快照
	snapshot := CacheSnapshot{
		Timestamp: time.Now(),
	}

	// 这里应该从实际的缓存服务获取统计信息
	// 简化实现，使用模拟数据
	snapshot.TotalRequests = cm.stats.TotalRequests + 100
	snapshot.HitRequests = cm.stats.HitRequests + 80
	snapshot.MissRequests = cm.stats.MissRequests + 20
	snapshot.ErrorRequests = cm.stats.ErrorRequests + 2

	// 计算比率
	if snapshot.TotalRequests > 0 {
		snapshot.HitRate = float64(snapshot.HitRequests) / float64(snapshot.TotalRequests)
		snapshot.MissRate = float64(snapshot.MissRequests) / float64(snapshot.TotalRequests)
		snapshot.ErrorRate = float64(snapshot.ErrorRequests) / float64(snapshot.TotalRequests)
	}

	snapshot.AvgResponseTime = time.Millisecond * 5
	snapshot.MemoryUsage = 1024 * 1024 * 100 // 100MB
	snapshot.ActiveConnections = 10

	// 更新统计
	cm.updateStats(snapshot)

	// 检查告警
	if cm.config.EnableAlerts {
		cm.checkAlerts(snapshot)
	}

	// 记录指标
	duration := time.Since(start)
	cm.recordMetrics("monitor", duration, true)
}

// updateStats 更新统计
func (cm *CacheMonitor) updateStats(snapshot CacheSnapshot) {
	cm.stats.TotalRequests = snapshot.TotalRequests
	cm.stats.HitRequests = snapshot.HitRequests
	cm.stats.MissRequests = snapshot.MissRequests
	cm.stats.ErrorRequests = snapshot.ErrorRequests
	cm.stats.HitRate = snapshot.HitRate
	cm.stats.MissRate = snapshot.MissRate
	cm.stats.ErrorRate = snapshot.ErrorRate
	cm.stats.AvgResponseTime = snapshot.AvgResponseTime
	cm.stats.MemoryUsage = snapshot.MemoryUsage
	cm.stats.ActiveConnections = snapshot.ActiveConnections

	// 添加到历史记录
	cm.stats.History = append(cm.stats.History, snapshot)

	// 保持历史记录大小
	if len(cm.stats.History) > cm.config.MaxHistorySize {
		cm.stats.History = cm.stats.History[1:]
	}
}

// checkAlerts 检查告警条件
func (cm *CacheMonitor) checkAlerts(snapshot CacheSnapshot) {
	alerts := []CacheAlert{}

	// 检查命中率告警
	if cm.config.AlertThresholds.HitRateMin > 0 && snapshot.HitRate < cm.config.AlertThresholds.HitRateMin {
		alerts = append(alerts, CacheAlert{
			ID:        generateAlertID(),
			Type:      "low_hit_rate",
			Message:   fmt.Sprintf("命中率过低: %.2f%% (阈值: %.2f%%)", snapshot.HitRate*100, cm.config.AlertThresholds.HitRateMin*100),
			Severity:  "warning",
			Timestamp: snapshot.Timestamp,
			Value:     snapshot.HitRate,
			Threshold: cm.config.AlertThresholds.HitRateMin,
			Resolved:  false,
		})
	}

	// 检查响应时间告警
	if cm.config.AlertThresholds.ResponseTimeMax > 0 && snapshot.AvgResponseTime > cm.config.AlertThresholds.ResponseTimeMax {
		alerts = append(alerts, CacheAlert{
			ID:        generateAlertID(),
			Type:      "high_response_time",
			Message:   fmt.Sprintf("响应时间过长: %v (阈值: %v)", snapshot.AvgResponseTime, cm.config.AlertThresholds.ResponseTimeMax),
			Severity:  "warning",
			Timestamp: snapshot.Timestamp,
			Value:     float64(snapshot.AvgResponseTime.Nanoseconds()) / 1e6,
			Threshold: float64(cm.config.AlertThresholds.ResponseTimeMax.Nanoseconds()) / 1e6,
			Resolved:  false,
		})
	}

	// 检查错误率告警
	if cm.config.AlertThresholds.ErrorRateMax > 0 && snapshot.ErrorRate > cm.config.AlertThresholds.ErrorRateMax {
		alerts = append(alerts, CacheAlert{
			ID:        generateAlertID(),
			Type:      "high_error_rate",
			Message:   fmt.Sprintf("错误率过高: %.2f%% (阈值: %.2f%%)", snapshot.ErrorRate*100, cm.config.AlertThresholds.ErrorRateMax*100),
			Severity:  "error",
			Timestamp: snapshot.Timestamp,
			Value:     snapshot.ErrorRate,
			Threshold: cm.config.AlertThresholds.ErrorRateMax,
			Resolved:  false,
		})
	}

	// 发送告警
	for _, alert := range alerts {
		cm.alerter.SendAlert(alert)
	}
}

// recordMetrics 记录指标
func (cm *CacheMonitor) recordMetrics(operation string, duration time.Duration, success bool) {
	if !cm.config.EnableMetrics {
		return
	}

	cm.metricsCollector.RecordDBQuery("cache_monitor", operation, duration, success)
	if !success {
		cm.metricsCollector.RecordDBError("cache_monitor_error", operation)
	}
}

// GetStats 获取统计信息
func (cm *CacheMonitor) GetStats() *CacheStats {
	return cm.stats
}

// GetMetrics 获取指标
func (cm *CacheMonitor) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// 基本指标
	metrics["total_requests"] = cm.stats.TotalRequests
	metrics["hit_rate"] = cm.stats.HitRate
	metrics["miss_rate"] = cm.stats.MissRate
	metrics["error_rate"] = cm.stats.ErrorRate
	metrics["avg_response_time"] = cm.stats.AvgResponseTime.String()
	metrics["memory_usage"] = cm.stats.MemoryUsage
	metrics["active_connections"] = cm.stats.ActiveConnections

	// 计算性能分数
	metrics["performance_score"] = cm.calculatePerformanceScore()

	// 历史趋势
	if len(cm.stats.History) > 1 {
		latest := cm.stats.History[len(cm.stats.History)-1]
		oldest := cm.stats.History[0]

		metrics["hit_rate_trend"] = latest.HitRate - oldest.HitRate
		metrics["response_time_trend"] = float64(latest.AvgResponseTime.Nanoseconds()) / float64(oldest.AvgResponseTime.Nanoseconds())
	}

	return metrics
}

// calculatePerformanceScore 计算性能分数
func (cm *CacheMonitor) calculatePerformanceScore() float64 {
	score := 100.0

	// 命中率影响 (40%)
	hitRateScore := cm.stats.HitRate * 40
	score -= (40 - hitRateScore)

	// 响应时间影响 (30%)
	if cm.stats.AvgResponseTime > time.Millisecond*10 {
		score -= 30
	} else if cm.stats.AvgResponseTime > time.Millisecond*5 {
		score -= 15
	}

	// 错误率影响 (20%)
	errorRateScore := cm.stats.ErrorRate * 20
	score -= errorRateScore

	// 内存使用影响 (10%)
	if cm.stats.MemoryUsage > 1024*1024*500 { // 500MB
		score -= 10
	} else if cm.stats.MemoryUsage > 1024*1024*200 { // 200MB
		score -= 5
	}

	if score < 0 {
		score = 0
	}

	return score
}

// GetAlerts 获取告警信息
func (cm *CacheMonitor) GetAlerts() []CacheAlert {
	return cm.alerter.GetAlerts()
}

// GenerateReport 生成报告
func (cm *CacheMonitor) GenerateReport(ctx context.Context, duration time.Duration) (*CacheReport, error) {
	if !cm.config.EnableReporting {
		return nil, fmt.Errorf("reporting is disabled")
	}

	endTime := time.Now()
	startTime := endTime.Add(-duration)

	report := &CacheReport{
		Period:    duration,
		StartTime: startTime,
		EndTime:   endTime,
		Alerts:    cm.alerter.GetAlerts(),
	}

	// 计算摘要
	report.Summary = cm.calculateSummary()

	// 生成详细信息
	report.Details = cm.generateDetails()

	// 生成建议
	report.Recommendations = cm.generateRecommendations()

	return report, nil
}

// calculateSummary 计算摘要
func (cm *CacheMonitor) calculateSummary() CacheSummary {
	return CacheSummary{
		TotalRequests:    cm.stats.TotalRequests,
		HitRate:          cm.stats.HitRate,
		AvgResponseTime:  cm.stats.AvgResponseTime,
		ErrorRate:        cm.stats.ErrorRate,
		MemoryUsage:      cm.stats.MemoryUsage,
		PerformanceScore: cm.calculatePerformanceScore(),
	}
}

// generateDetails 生成详细信息
func (cm *CacheMonitor) generateDetails() ReportDetails {
	details := ReportDetails{
		TopKeys:           cm.getTopKeys(),
		ResponseTimeStats: cm.getResponseTimeStats(),
		ErrorStats:        cm.getErrorStats(),
		Trends:            cm.getTrends(),
	}

	return details
}

// getTopKeys 获取热门键
func (cm *CacheMonitor) getTopKeys() []KeyStats {
	// 简化实现，返回模拟数据
	return []KeyStats{
		{
			Key:         "user:1",
			AccessCount: 1000,
			HitCount:    850,
			HitRate:     0.85,
			LastAccess:  time.Now(),
		},
		{
			Key:         "config:app",
			AccessCount: 500,
			HitCount:    450,
			HitRate:     0.9,
			LastAccess:  time.Now().Add(-time.Minute * 5),
		},
	}
}

// getResponseTimeStats 获取响应时间统计
func (cm *CacheMonitor) getResponseTimeStats() []TimeStats {
	return []TimeStats{
		{
			Percentile: "P50",
			Value:      time.Millisecond * 3,
		},
		{
			Percentile: "P95",
			Value:      time.Millisecond * 8,
		},
		{
			Percentile: "P99",
			Value:      time.Millisecond * 15,
		},
	}
}

// getErrorStats 获取错误统计
func (cm *CacheMonitor) getErrorStats() []ErrorStats {
	return []ErrorStats{
		{
			ErrorType: "connection_error",
			Count:     5,
			LastSeen:  time.Now().Add(-time.Minute * 10),
		},
		{
			ErrorType: "timeout_error",
			Count:     2,
			LastSeen:  time.Now().Add(-time.Minute * 30),
		},
	}
}

// getTrends 获取趋势数据
func (cm *CacheMonitor) getTrends() []TrendData {
	trends := make([]TrendData, 0)

	// 简化实现，返回模拟趋势数据
	now := time.Now()
	for i := 0; i < 24; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Hour)
		trends = append(trends, TrendData{
			Timestamp: timestamp,
			Metric:    "hit_rate",
			Value:     0.85 + float64(i%10)*0.01, // 模拟数据
		})
	}

	return trends
}

// generateRecommendations 生成建议
func (cm *CacheMonitor) generateRecommendations() []string {
	recommendations := []string{}

	// 基于性能分数生成建议
	score := cm.calculatePerformanceScore()
	if score < 80 {
		recommendations = append(recommendations, "缓存性能较低，建议优化缓存策略")
	}

	// 基于命中率生成建议
	if cm.stats.HitRate < 0.8 {
		recommendations = append(recommendations, "命中率较低，建议增加缓存预热或调整缓存策略")
	}

	// 基于响应时间生成建议
	if cm.stats.AvgResponseTime > time.Millisecond*10 {
		recommendations = append(recommendations, "响应时间较长，建议优化缓存实现或增加缓存容量")
	}

	// 基于错误率生成建议
	if cm.stats.ErrorRate > 0.05 {
		recommendations = append(recommendations, "错误率较高，建议检查缓存配置和网络连接")
	}

	// 基于内存使用生成建议
	if cm.stats.MemoryUsage > 1024*1024*500 {
		recommendations = append(recommendations, "内存使用较高，建议清理过期缓存或增加缓存容量")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "缓存运行良好，继续保持当前配置")
	}

	return recommendations
}

// SendAlert 发送告警
func (ca *CacheAlerter) SendAlert(alert CacheAlert) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.alerts = append(ca.alerts, alert)

	// 保持最近1000条告警
	if len(ca.alerts) > 1000 {
		ca.alerts = ca.alerts[len(ca.alerts)-1000:]
	}

	// 这里可以实现实际的告警发送逻辑
	// 例如发送邮件、Slack、短信等
	log.Printf("Cache Alert [%s]: %s (Value: %.2f, Threshold: %.2f)",
		alert.Severity, alert.Message, alert.Value, alert.Threshold)
}

// GetAlerts 获取告警
func (ca *CacheAlerter) GetAlerts() []CacheAlert {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	alerts := make([]CacheAlert, len(ca.alerts))
	copy(alerts, ca.alerts)
	return alerts
}

// ResolveAlert 解决告警
func (ca *CacheAlerter) ResolveAlert(alertID string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	for i, alert := range ca.alerts {
		if alert.ID == alertID {
			ca.alerts[i].Resolved = true
			return nil
		}
	}

	return fmt.Errorf("alert %s not found", alertID)
}

// ExportReport 导出报告
func (cr *CacheReporter) ExportReport(report *CacheReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}

	return string(data), nil
}

// generateAlertID 生成告警ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// ResetStats 重置统计
func (cm *CacheMonitor) ResetStats() {
	cm.stats = &CacheStats{
		History:   make([]CacheSnapshot, 0),
		LastReset: time.Now(),
	}

	// 记录重置指标
	cm.recordMetrics("stats_reset", time.Millisecond*10, true)
}

// GetHealthStatus 获取健康状态
func (cm *CacheMonitor) GetHealthStatus() map[string]interface{} {
	status := make(map[string]interface{})

	score := cm.calculatePerformanceScore()

	status["status"] = "healthy"
	status["score"] = score
	status["message"] = "Cache is operating normally"

	if score < 60 {
		status["status"] = "unhealthy"
		status["message"] = "Cache performance is poor"
	} else if score < 80 {
		status["status"] = "degraded"
		status["message"] = "Cache performance is degraded"
	}

	status["metrics"] = cm.GetMetrics()
	status["alerts"] = cm.GetAlerts()

	return status
}

// Close 关闭监控器
func (cm *CacheMonitor) Close() error {
	// 清理资源
	return nil
}
