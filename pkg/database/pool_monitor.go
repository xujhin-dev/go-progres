package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"

	"gorm.io/gorm"
)

// PoolMonitor 连接池监控器
type PoolMonitor struct {
	db               *gorm.DB
	metricsCollector *metrics.MetricsCollector
	stats            *PoolStats
	config           *PoolMonitorConfig
	stopCh           chan struct{}
	mu               sync.RWMutex
}

// PoolMonitorConfig 连接池监控配置
type PoolMonitorConfig struct {
	MonitorInterval  time.Duration `json:"monitor_interval"`
	AlertThreshold   int           `json:"alert_threshold"`
	MaxHistorySize   int           `json:"max_history_size"`
	EnableAutoTuning bool          `json:"enable_auto_tuning"`
	EnableAlerts     bool          `json:"enable_alerts"`
}

// PoolStats 连接池统计
type PoolStats struct {
	OpenConnections   int            `json:"open_connections"`
	InUse             int            `json:"in_use"`
	Idle              int            `json:"idle"`
	WaitCount         int64          `json:"wait_count"`
	WaitDuration      time.Duration  `json:"wait_duration"`
	MaxIdleClosed     int64          `json:"max_idle_closed"`
	MaxLifetimeClosed int64          `json:"max_lifetime_closed"`
	LastReset         time.Time      `json:"last_reset"`
	History           []PoolSnapshot `json:"history"`
}

// PoolSnapshot 连接池快照
type PoolSnapshot struct {
	Timestamp         time.Time     `json:"timestamp"`
	OpenConnections   int           `json:"open_connections"`
	InUse             int           `json:"in_use"`
	Idle              int           `json:"idle"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	MaxConnections     int           `json:"max_connections"`
	MaxWaitTime        time.Duration `json:"max_wait_time"`
	MaxIdleTime        time.Duration `json:"max_idle_time"`
	MinIdleConnections int           `json:"min_idle_connections"`
}

// PoolAlert 连接池告警
type PoolAlert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Resolved  bool      `json:"resolved"`
}

// PoolTuner 连接池调优器
type PoolTuner struct {
	db     *gorm.DB
	config *PoolTunerConfig
}

// PoolTunerConfig 连接池调优配置
type PoolTunerConfig struct {
	EnableAutoTuning   bool          `json:"enable_auto_tuning"`
	TuningInterval     time.Duration `json:"tuning_interval"`
	MaxOpenConnections int           `json:"max_open_connections"`
	MaxIdleConnections int           `json:"max_idle_connections"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time"`
}

// NewPoolMonitor 创建连接池监控器
func NewPoolMonitor(db *gorm.DB, metricsCollector *metrics.MetricsCollector, config *PoolMonitorConfig) *PoolMonitor {
	pm := &PoolMonitor{
		db:               db,
		metricsCollector: metricsCollector,
		stats:            &PoolStats{History: make([]PoolSnapshot, 0)},
		config:           config,
		stopCh:           make(chan struct{}),
	}

	// 启动监控
	go pm.startMonitoring()

	return pm
}

// startMonitoring 开始监控
func (pm *PoolMonitor) startMonitoring() {
	ticker := time.NewTicker(pm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.collectStats()
		case <-pm.stopCh:
			return
		}
	}
}

// collectStats 收集统计信息
func (pm *PoolMonitor) collectStats() {
	sqlDB, err := pm.db.DB()
	if err != nil {
		log.Printf("Failed to get database connection: %v", err)
		return
	}

	stats := sqlDB.Stats()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 创建快照
	snapshot := PoolSnapshot{
		Timestamp:         time.Now(),
		OpenConnections:   stats.OpenConnections,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}

	// 更新当前统计
	pm.stats.OpenConnections = stats.OpenConnections
	pm.stats.InUse = stats.InUse
	pm.stats.Idle = stats.Idle
	pm.stats.WaitCount = stats.WaitCount
	pm.stats.WaitDuration = stats.WaitDuration
	pm.stats.MaxIdleClosed = stats.MaxIdleClosed
	pm.stats.MaxLifetimeClosed = stats.MaxLifetimeClosed

	// 添加到历史记录
	pm.stats.History = append(pm.stats.History, snapshot)

	// 保持历史记录大小
	if len(pm.stats.History) > pm.config.MaxHistorySize {
		pm.stats.History = pm.stats.History[1:]
	}

	// 检查告警条件
	if pm.config.EnableAlerts {
		pm.checkAlerts(snapshot)
	}

	// 自动调优
	if pm.config.EnableAutoTuning {
		pm.autoTune()
	}

	// 记录指标
	pm.recordMetrics(snapshot)
}

// checkAlerts 检查告警条件
func (pm *PoolMonitor) checkAlerts(snapshot PoolSnapshot) {
	alerts := []PoolAlert{}

	// 检查连接数告警
	if snapshot.OpenConnections > pm.config.AlertThreshold {
		alerts = append(alerts, PoolAlert{
			ID:        generateAlertID(),
			Type:      "high_connections",
			Message:   fmt.Sprintf("连接数过高: %d (阈值: %d)", snapshot.OpenConnections, pm.config.AlertThreshold),
			Severity:  "warning",
			Timestamp: snapshot.Timestamp,
			Value:     float64(snapshot.OpenConnections),
			Threshold: float64(pm.config.AlertThreshold),
			Resolved:  false,
		})
	}

	// 检查等待时间告警
	if snapshot.WaitDuration > time.Second*5 {
		alerts = append(alerts, PoolAlert{
			ID:        generateAlertID(),
			Type:      "high_wait_time",
			Message:   fmt.Sprintf("等待时间过长: %v (阈值: 5s)", snapshot.WaitDuration),
			Severity:  "warning",
			Timestamp: snapshot.Timestamp,
			Value:     float64(snapshot.WaitDuration.Seconds()),
			Threshold: 5.0,
			Resolved:  false,
		})
	}

	// 检查空闲连接告警
	if snapshot.Idle < 5 {
		alerts = append(alerts, PoolAlert{
			ID:        generateAlertID(),
			Type:      "low_idle_connections",
			Message:   fmt.Sprintf("空闲连接过少: %d", snapshot.Idle),
			Severity:  "warning",
			Timestamp: snapshot.Timestamp,
			Value:     float64(snapshot.Idle),
			Threshold: 5.0,
			Resolved:  false,
		})
	}

	// 发送告警
	for _, alert := range alerts {
		pm.sendAlert(alert)
	}
}

// sendAlert 发送告警
func (pm *PoolMonitor) sendAlert(alert PoolAlert) {
	// 记录告警指标
	pm.metricsCollector.RecordDBError("pool_alert", alert.Type)

	// 这里可以实现实际的告警发送逻辑
	// 例如发送邮件、Slack、短信等
	log.Printf("Pool Alert [%s]: %s (Value: %.2f, Threshold: %.2f)",
		alert.Severity, alert.Message, alert.Value, alert.Threshold)
}

// autoTune 自动调优连接池
func (pm *PoolMonitor) autoTune() {
	// 简化的自动调优逻辑
	// 实际项目中应该基于历史数据和机器学习算法进行调优

	sqlDB, err := pm.db.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()

	// 建议的配置
	suggestedConfig := &PoolTunerConfig{
		MaxOpenConnections: pm.calculateOptimalMaxOpen(stats),
		MaxIdleConnections: pm.calculateOptimalIdle(stats),
		ConnMaxLifetime:    time.Hour * 2,
		ConnMaxIdleTime:    time.Minute * 30,
	}

	// 应用调优建议
	if stats.MaxOpenConnections != suggestedConfig.MaxOpenConnections {
		log.Printf("建议调整 MaxOpenConnections: %d -> %d",
			stats.MaxOpenConnections, suggestedConfig.MaxOpenConnections)
	}
}

// calculateOptimalMaxOpen 计算最优的最大连接数
func (pm *PoolMonitor) calculateOptimalMaxOpen(stats sql.DBStats) int {
	// 基于当前使用情况计算最优连接数
	// 简化实现：基于使用率计算

	usageRate := float64(stats.InUse) / float64(stats.OpenConnections)

	if usageRate > 0.8 {
		// 使用率高，增加连接数
		return int(float64(stats.OpenConnections) * 1.2)
	} else if usageRate < 0.3 && stats.OpenConnections > 10 {
		// 使用率低，减少连接数
		return int(float64(stats.OpenConnections) * 0.8)
	}

	return stats.OpenConnections
}

// calculateOptimalIdle 计算最优的空闲连接数
func (pm *PoolMonitor) calculateOptimalIdle(stats sql.DBStats) int {
	// 基于当前使用情况计算最优空闲连接数
	// 简化实现：基于使用率和连接数计算

	usageRate := float64(stats.InUse) / float64(stats.OpenConnections)

	if usageRate > 0.7 {
		// 使用率高，减少空闲连接
		return int(float64(stats.OpenConnections) * 0.2)
	} else if usageRate < 0.3 {
		// 使用率低，增加空闲连接
		return int(float64(stats.OpenConnections) * 0.5)
	}

	return int(float64(stats.OpenConnections) * 0.3)
}

// recordMetrics 记录指标
func (pm *PoolMonitor) recordMetrics(snapshot PoolSnapshot) {
	// 记录连接池指标
	pm.metricsCollector.UpdateActiveGoroutines(snapshot.OpenConnections)
	pm.metricsCollector.UpdateMemoryUsage(int(snapshot.OpenConnections * 1024)) // 估算内存使用

	// 记录等待时间指标
	pm.metricsCollector.RecordDBQuery("pool_wait_time", "connection", snapshot.WaitDuration, true)

	// 记录连接数指标
	pm.metricsCollector.RecordDBError("pool_connections", "open")
}

// GetStats 获取连接池统计
func (pm *PoolMonitor) GetStats() *PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 返回统计信息的副本
	return &PoolStats{
		OpenConnections:   pm.stats.OpenConnections,
		InUse:             pm.stats.InUse,
		Idle:              pm.stats.Idle,
		WaitCount:         pm.stats.WaitCount,
		WaitDuration:      pm.stats.WaitDuration,
		MaxIdleClosed:     pm.stats.MaxIdleClosed,
		MaxLifetimeClosed: pm.stats.MaxLifetimeClosed,
		LastReset:         pm.stats.LastReset,
		History:           append([]PoolSnapshot{}, pm.stats.History...),
	}
}

// GetHistory 获取历史记录
func (pm *PoolMonitor) GetHistory(limit int) []PoolSnapshot {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if limit <= 0 || limit > len(pm.stats.History) {
		limit = len(pm.stats.History)
	}

	return append([]PoolSnapshot{}, pm.stats.History[len(pm.stats.History)-limit:]...)
}

// GetPerformanceMetrics 获取性能指标
func (pm *PoolMonitor) GetPerformanceMetrics() map[string]interface{} {
	stats := pm.GetStats()

	metrics := make(map[string]interface{})

	// 基本指标
	metrics["open_connections"] = stats.OpenConnections
	metrics["in_use"] = stats.InUse
	metrics["idle"] = stats.Idle
	metrics["wait_count"] = stats.WaitCount
	metrics["wait_duration"] = stats.WaitDuration.String()

	// 计算指标
	if stats.OpenConnections > 0 {
		metrics["utilization_rate"] = float64(stats.InUse) / float64(stats.OpenConnections)
		metrics["idle_rate"] = float64(stats.Idle) / float64(stats.OpenConnections)
	}

	// 历史趋势
	if len(stats.History) > 1 {
		latest := stats.History[len(stats.History)-1]
		oldest := stats.History[0]

		metrics["connection_trend"] = float64(latest.OpenConnections) / float64(oldest.OpenConnections)
		metrics["wait_time_trend"] = float64(latest.WaitDuration.Nanoseconds()) / float64(oldest.WaitDuration.Nanoseconds())
	}

	return metrics
}

// ResetStats 重置统计
func (pm *PoolMonitor) ResetStats() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.stats = &PoolStats{
		History:   make([]PoolSnapshot, 0),
		LastReset: time.Now(),
	}

	// 记录重置指标
	pm.metricsCollector.RecordDBError("pool_stats_reset", "all")
}

// Close 关闭监控器
func (pm *PoolMonitor) Close() {
	close(pm.stopCh)
}

// NewPoolTuner 创建连接池调优器
func NewPoolTuner(db *gorm.DB, config *PoolTunerConfig) *PoolTuner {
	return &PoolTuner{
		db:     db,
		config: config,
	}
}

// Tune 调优连接池
func (pt *PoolTuner) Tune() error {
	sqlDB, err := pt.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// 应用调优配置
	if pt.config.MaxOpenConnections > 0 {
		sqlDB.SetMaxOpenConns(pt.config.MaxOpenConnections)
	}

	if pt.config.MaxIdleConnections > 0 {
		sqlDB.SetMaxIdleConns(pt.config.MaxIdleConnections)
	}

	if pt.config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(pt.config.ConnMaxLifetime)
	}

	if pt.config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(pt.config.ConnMaxIdleTime)
	}

	// 记录调优指标
	log.Printf("Tuned connection pool: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		pt.config.MaxOpenConnections, pt.config.MaxIdleConnections,
		pt.config.ConnMaxLifetime, pt.config.ConnMaxIdleTime)

	return nil
}

// ValidateConfig 验证配置
func (pt *PoolTuner) ValidateConfig() error {
	if pt.config.MaxOpenConnections <= 0 {
		return fmt.Errorf("MaxOpenConnections must be positive")
	}

	if pt.config.MaxIdleConnections < 0 {
		return fmt.Errorf("MaxIdleConnections cannot be negative")
	}

	if pt.config.MaxIdleConnections > pt.config.MaxOpenConnections {
		return fmt.Errorf("MaxIdleConnections cannot be greater than MaxOpenConnections")
	}

	if pt.config.ConnMaxLifetime <= 0 {
		return fmt.Errorf("ConnMaxLifetime must be positive")
	}

	if pt.config.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("ConnMaxIdleTime must be positive")
	}

	return nil
}

// GetTuningRecommendations 获取调优建议
func (pm *PoolMonitor) GetTuningRecommendations() []TuningRecommendation {
	stats := pm.GetStats()

	var recommendations []TuningRecommendation

	// 连接数建议
	if stats.OpenConnections > 100 {
		recommendations = append(recommendations, TuningRecommendation{
			Type:        "connections",
			Current:     stats.OpenConnections,
			Recommended: 50,
			Reason:      "连接数过多可能导致资源浪费",
			Priority:    "medium",
			Impact:      "performance",
		})
	} else if stats.OpenConnections < 10 && stats.InUse > 5 {
		recommendations = append(recommendations, TuningRecommendation{
			Type:        "connections",
			Current:     stats.OpenConnections,
			Recommended: 20,
			Reason:      "连接数不足可能导致等待",
			Priority:    "high",
			Impact:      "performance",
		})
	}

	// 空闲连接建议
	if stats.Idle < 5 {
		recommendations = append(recommendations, TuningRecommendation{
			Type:        "idle_connections",
			Current:     stats.Idle,
			Recommended: 10,
			Reason:      "空闲连接过少可能导致连接建立开销",
			Priority:    "medium",
			Impact:      "performance",
		})
	} else if stats.Idle > stats.OpenConnections/2 {
		recommendations = append(recommendations, TuningRecommendation{
			Type:        "idle_connections",
			Current:     stats.Idle,
			Recommended: stats.OpenConnections / 3,
			Reason:      "空闲连接过多可能导致内存浪费",
			Priority:    "low",
			Impact:      "memory",
		})
	}

	// 等待时间建议
	if stats.WaitDuration > time.Second*2 {
		recommendations = append(recommendations, TuningRecommendation{
			Type:        "wait_time",
			Current:     stats.WaitDuration.String(),
			Recommended: "1s",
			Reason:      "等待时间过长影响用户体验",
			Priority:    "high",
			Impact:      "user_experience",
		})
	}

	return recommendations
}

// TuningRecommendation 调优建议
type TuningRecommendation struct {
	Type        string      `json:"type"`
	Current     interface{} `json:"current"`
	Recommended interface{} `json:"recommended"`
	Reason      string      `json:"reason"`
	Priority    string      `json:"priority"`
	Impact      string      `json:"impact"`
}

// generateAlertID 生成告警ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// StartAutoTuning 启动自动调优
func (pm *PoolMonitor) StartAutoTuning(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if pm.config.EnableAutoTuning {
				pm.autoTune()
			}
		case <-pm.stopCh:
			return
		}
	}
}

// HealthCheck 健康检查
func (pm *PoolMonitor) HealthCheck() error {
	sqlDB, err := pm.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// 测试连接
	err = sqlDB.Ping()
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// 检查基本指标
	stats := sqlDB.Stats()
	if stats.OpenConnections == 0 {
		return fmt.Errorf("no database connections available")
	}

	return nil
}

// ExportMetrics 导出指标
func (pm *PoolMonitor) ExportMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// 当前指标
	currentStats := pm.GetStats()
	metrics["current"] = currentStats

	// 性能指标
	metrics["performance"] = pm.GetPerformanceMetrics()

	// 历史趋势
	if len(currentStats.History) > 0 {
		metrics["trends"] = pm.calculateTrends(currentStats.History)
	}

	// 调优建议
	metrics["recommendations"] = pm.GetTuningRecommendations()

	return metrics, nil
}

// calculateTrends 计算趋势
func (pm *PoolMonitor) calculateTrends(history []PoolSnapshot) map[string]interface{} {
	if len(history) < 2 {
		return map[string]interface{}{}
	}

	trends := make(map[string]interface{})

	oldest := history[0]
	latest := history[len(history)-1]

	// 连接数趋势
	if oldest.OpenConnections > 0 {
		trends["connections_trend"] = float64(latest.OpenConnections) / float64(oldest.OpenConnections)
	}

	// 等待时间趋势
	if oldest.WaitDuration.Nanoseconds() > 0 {
		trends["wait_time_trend"] = float64(latest.WaitDuration.Nanoseconds()) / float64(oldest.WaitDuration.Nanoseconds())
	}

	// 使用率趋势
	if oldest.OpenConnections > 0 {
		oldUtilization := float64(oldest.InUse) / float64(oldest.OpenConnections)
		newUtilization := float64(latest.InUse) / float64(latest.OpenConnections)
		trends["utilization_trend"] = newUtilization / oldUtilization
	}

	return trends
}
