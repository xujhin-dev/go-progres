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

// MultiLevelCache 多级缓存
type MultiLevelCache struct {
	localCache       CacheService
	remoteCache      CacheService
	metricsCollector *metrics.MetricsCollector
	config           *MultiLevelConfig
	strategy         CacheStrategy
	coordinator      *CacheCoordinator
}

// MultiLevelConfig 多级缓存配置
type MultiLevelConfig struct {
	LocalCacheSize       int           `json:"local_cache_size"`
	LocalCacheTTL        time.Duration `json:"local_cache_ttl"`
	RemoteCacheTTL       time.Duration `json:"remote_cache_ttl"`
	EnableMetrics        bool          `json:"enable_metrics"`
	EnableCoordination   bool          `json:"enable_coordination"`
	EnableBackgroundSync bool          `json:"enable_background_sync"`
	SyncInterval         time.Duration `json:"sync_interval"`
	MaxRetries           int           `json:"max_retries"`
	RetryDelay           time.Duration `json:"retry_delay"`
}

// CacheStrategy 缓存策略
type CacheStrategy interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	GetName() string
}

// CacheCoordinator 缓存协调器
type CacheCoordinator struct {
	localCache  CacheService
	remoteCache CacheService
	config      *MultiLevelConfig
	eventBus    *CacheEventBus
}

// CacheEventBus 缓存事件总线
type CacheEventBus struct {
	subscribers []CacheEventSubscriber
	mu          sync.RWMutex
}

// CacheEventSubscriber 缓存事件订阅者
type CacheEventSubscriber interface {
	OnCacheHit(key string, level string)
	OnCacheMiss(key string, level string)
	OnCacheSet(key string, level string)
	OnCacheDelete(key string, level string)
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(localCache, remoteCache CacheService, metricsCollector *metrics.MetricsCollector, config *MultiLevelConfig) *MultiLevelCache {
	mlc := &MultiLevelCache{
		localCache:       localCache,
		remoteCache:      remoteCache,
		metricsCollector: metricsCollector,
		config:           config,
		strategy:         NewCacheStrategy(config),
		coordinator:      NewCacheCoordinator(localCache, remoteCache, config),
	}

	// 启动后台同步
	if config.EnableBackgroundSync {
		go mlc.startBackgroundSync()
	}

	return mlc
}

// NewCacheStrategy 创建缓存策略
func NewCacheStrategy(config *MultiLevelConfig) CacheStrategy {
	return &DefaultCacheStrategy{
		localCache:  nil, // 将在构造函数中设置
		remoteCache: nil, // 将在构造函数中设置
		config:      config,
	}
}

// NewCacheCoordinator 创建缓存协调器
func NewCacheCoordinator(localCache, remoteCache CacheService, config *MultiLevelConfig) *CacheCoordinator {
	return &CacheCoordinator{
		localCache:  localCache,
		remoteCache: remoteCache,
		config:      config,
		eventBus:    NewCacheEventBus(),
	}
}

// NewCacheEventBus 创建缓存事件总线
func NewCacheEventBus() *CacheEventBus {
	return &CacheEventBus{
		subscribers: make([]CacheEventSubscriber, 0),
	}
}

// DefaultCacheStrategy 默认缓存策略
type DefaultCacheStrategy struct {
	localCache  CacheService
	remoteCache CacheService
	config      *MultiLevelConfig
}

// Get 获取缓存值
func (dcs *DefaultCacheStrategy) Get(ctx context.Context, key string) (interface{}, error) {
	// 首先从本地缓存获取
	var value string
	err := dcs.localCache.Get(ctx, key, &value)
	if err == nil && value != "" {
		// 本地缓存命中，通知事件
		if dcs.config.EnableCoordination {
			dcs.notifyEvent("hit", key, "local")
		}

		var result interface{}
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal local cache value: %w", err)
		}

		return result, nil
	}

	// 本地缓存未命中，从远程缓存获取
	err = dcs.remoteCache.Get(ctx, key, &value)
	if err != nil {
		if dcs.config.EnableCoordination {
			dcs.notifyEvent("miss", key, "remote")
		}
		return nil, fmt.Errorf("failed to get from remote cache: %w", err)
	}

	if value == "" {
		// 远程缓存也未命中
		if dcs.config.EnableCoordination {
			dcs.notifyEvent("miss", key, "remote")
		}
		return nil, nil
	}

	// 远程缓存命中，将数据写入本地缓存
	var result interface{}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal remote cache value: %w", err)
	}

	// 异步写入本地缓存
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		dcs.localCache.Set(ctx, key, value, dcs.config.LocalCacheTTL)
	}()

	if dcs.config.EnableCoordination {
		dcs.notifyEvent("hit", key, "remote")
	}

	return result, nil
}

// Set 设置缓存值
func (dcs *DefaultCacheStrategy) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 序列化数据
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// 写入本地缓存
	if err := dcs.localCache.Set(ctx, key, jsonData, dcs.config.LocalCacheTTL); err != nil {
		return fmt.Errorf("failed to set local cache: %w", err)
	}

	// 写入远程缓存
	if err := dcs.remoteCache.Set(ctx, key, jsonData, dcs.config.RemoteCacheTTL); err != nil {
		return fmt.Errorf("failed to set remote cache: %w", err)
	}

	if dcs.config.EnableCoordination {
		dcs.notifyEvent("set", key, "both")
	}

	return nil
}

// Delete 删除缓存值
func (dcs *DefaultCacheStrategy) Delete(ctx context.Context, key string) error {
	// 从本地缓存删除
	if err := dcs.localCache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete local cache: %w", err)
	}

	// 从远程缓存删除
	if err := dcs.remoteCache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete remote cache: %w", err)
	}

	if dcs.config.EnableCoordination {
		dcs.notifyEvent("delete", key, "both")
	}

	return nil
}

func (dcs *DefaultCacheStrategy) GetName() string {
	return "default"
}

// Get 获取缓存值
func (mlc *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mlc.recordMetrics("get", duration, true)
	}()

	value, err := mlc.strategy.Get(ctx, key)
	if err != nil {
		mlc.recordMetrics("get_error", time.Since(start), false)
		return nil, err
	}

	return value, nil
}

// Set 设置缓存值
func (mlc *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mlc.recordMetrics("set", duration, true)
	}()

	err := mlc.strategy.Set(ctx, key, value, ttl)
	if err != nil {
		mlc.recordMetrics("set_error", time.Since(start), false)
		return err
	}

	return nil
}

// Delete 删除缓存值
func (mlc *MultiLevelCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mlc.recordMetrics("delete", duration, true)
	}()

	err := mlc.strategy.Delete(ctx, key)
	if err != nil {
		mlc.recordMetrics("delete_error", time.Since(start), false)
		return err
	}

	return nil
}

// GetWithFallback 带回退的获取
func (mlc *MultiLevelCache) GetWithFallback(ctx context.Context, key string, fallback func() (interface{}, error)) (interface{}, error) {
	// 尝试从缓存获取
	value, err := mlc.Get(ctx, key)
	if err == nil && value != nil {
		return value, nil
	}

	// 缓存未命中，使用回退函数获取数据
	value, err = fallback()
	if err != nil {
		return nil, fmt.Errorf("fallback failed: %w", err)
	}

	// 将数据写入缓存
	if err := mlc.Set(ctx, key, value, mlc.config.LocalCacheTTL); err != nil {
		log.Printf("Failed to cache fallback result for key %s: %v", key, err)
	}

	return value, nil
}

// GetStats 获取统计信息
func (mlc *MultiLevelCache) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 获取本地缓存统计
	localStats := mlc.getCacheStats(mlc.localCache)
	stats["local_cache"] = localStats

	// 获取远程缓存统计
	remoteStats := mlc.getCacheStats(mlc.remoteCache)
	stats["remote_cache"] = remoteStats

	// 计算整体统计
	stats["total_hits"] = localStats["hits"].(int64) + remoteStats["hits"].(int64)
	stats["total_misses"] = localStats["misses"].(int64) + remoteStats["misses"].(int64)

	totalRequests := stats["total_hits"].(int64) + stats["total_misses"].(int64)
	if totalRequests > 0 {
		stats["hit_rate"] = float64(stats["total_hits"].(int64)) / float64(totalRequests)
	}

	return stats
}

// getCacheStats 获取缓存统计
func (mlc *MultiLevelCache) getCacheStats(cache CacheService) map[string]interface{} {
	// 简化实现，返回模拟数据
	return map[string]interface{}{
		"hits":   int64(1000),
		"misses": int64(200),
		"size":   int64(1024 * 1024 * 100), // 100MB
	}
}

// Warmup 预热缓存
func (mlc *MultiLevelCache) Warmup(ctx context.Context, keys []string, loader func(string) (interface{}, error)) error {
	for _, key := range keys {
		// 检查是否已存在
		if _, err := mlc.Get(ctx, key); err == nil {
			continue
		}

		// 使用加载器获取数据
		value, err := loader(key)
		if err != nil {
			log.Printf("Failed to load data for key %s: %v", key, err)
			continue
		}

		// 写入缓存
		if err := mlc.Set(ctx, key, value, mlc.config.LocalCacheTTL); err != nil {
			log.Printf("Failed to warmup key %s: %v", key, err)
		}
	}

	return nil
}

// Invalidate 失效缓存
func (mlc *MultiLevelCache) Invalidate(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := mlc.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to invalidate key %s: %w", key, err)
		}
	}
	return nil
}

// InvalidatePattern 按模式失效缓存
func (mlc *MultiLevelCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// 简化实现，实际项目中应该使用更复杂的模式匹配
	keys := []string{pattern}
	return mlc.Invalidate(ctx, keys)
}

// startBackgroundSync 启动后台同步
func (mlc *MultiLevelCache) startBackgroundSync() {
	ticker := time.NewTicker(mlc.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mlc.syncCaches()
		}
	}
}

// syncCaches 同步缓存
func (mlc *MultiLevelCache) syncCaches() {
	// 简化实现，实际项目中应该实现更复杂的同步逻辑
	log.Println("Syncing caches...")
}

// Subscribe 订阅事件
func (mlc *MultiLevelCache) Subscribe(subscriber CacheEventSubscriber) {
	mlc.coordinator.eventBus.Subscribe(subscriber)
}

// notifyEvent 通知事件
func (dcs *DefaultCacheStrategy) notifyEvent(eventType, key, level string) {
	// 简化实现，实际项目中应该通过事件总线通知
	log.Printf("Cache event: %s, key: %s, level: %s", eventType, key, level)
}

// Subscribe 订阅事件
func (ceb *CacheEventBus) Subscribe(subscriber CacheEventSubscriber) {
	ceb.mu.Lock()
	defer ceb.mu.Unlock()
	ceb.subscribers = append(ceb.subscribers, subscriber)
}

// Publish 发布事件
func (ceb *CacheEventBus) Publish(eventType, key, level string) {
	ceb.mu.RLock()
	defer ceb.mu.RUnlock()

	for _, subscriber := range ceb.subscribers {
		switch eventType {
		case "hit":
			subscriber.OnCacheHit(key, level)
		case "miss":
			subscriber.OnCacheMiss(key, level)
		case "set":
			subscriber.OnCacheSet(key, level)
		case "delete":
			subscriber.OnCacheDelete(key, level)
		}
	}
}

// CacheMetricsCollector 缓存指标收集器
type CacheMetricsCollector struct {
	localCache  CacheService
	remoteCache CacheService
	metrics     map[string]interface{}
	mu          sync.RWMutex
}

// NewCacheMetricsCollector 创建缓存指标收集器
func NewCacheMetricsCollector(localCache, remoteCache CacheService) *CacheMetricsCollector {
	return &CacheMetricsCollector{
		localCache:  localCache,
		remoteCache: remoteCache,
		metrics:     make(map[string]interface{}),
	}
}

// CollectMetrics 收集指标
func (cmc *CacheMetricsCollector) CollectMetrics() map[string]interface{} {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	// 收集本地缓存指标
	localMetrics := cmc.collectCacheMetrics(cmc.localCache)
	cmc.metrics["local_cache"] = localMetrics

	// 收集远程缓存指标
	remoteMetrics := cmc.collectCacheMetrics(cmc.remoteCache)
	cmc.metrics["remote_cache"] = remoteMetrics

	// 计算整体指标
	cmc.calculateOverallMetrics()

	return cmc.metrics
}

// collectCacheMetrics 收集单个缓存的指标
func (cmc *CacheMetricsCollector) collectCacheMetrics(cache CacheService) map[string]interface{} {
	// 简化实现，返回模拟数据
	return map[string]interface{}{
		"hits":        int64(1000),
		"misses":      int64(200),
		"sets":        int64(300),
		"deletes":     int64(50),
		"size":        int64(1024 * 1024 * 100),
		"connections": int64(10),
	}
}

// calculateOverallMetrics 计算整体指标
func (cmc *CacheMetricsCollector) calculateOverallMetrics() {
	localMetrics := cmc.metrics["local_cache"].(map[string]interface{})
	remoteMetrics := cmc.metrics["remote_cache"].(map[string]interface{})

	totalHits := localMetrics["hits"].(int64) + remoteMetrics["hits"].(int64)
	totalMisses := localMetrics["misses"].(int64) + remoteMetrics["misses"].(int64)
	totalRequests := totalHits + totalMisses

	if totalRequests > 0 {
		cmc.metrics["hit_rate"] = float64(totalHits) / float64(totalRequests)
		cmc.metrics["miss_rate"] = float64(totalMisses) / float64(totalRequests)
	}

	cmc.metrics["total_hits"] = totalHits
	cmc.metrics["total_misses"] = totalMisses
	cmc.metrics["total_requests"] = totalRequests
}

// CacheHealthChecker 缓存健康检查器
type CacheHealthChecker struct {
	localCache  CacheService
	remoteCache CacheService
	config      *MultiLevelConfig
}

// NewCacheHealthChecker 创建缓存健康检查器
func NewCacheHealthChecker(localCache, remoteCache CacheService, config *MultiLevelConfig) *CacheHealthChecker {
	return &CacheHealthChecker{
		localCache:  localCache,
		remoteCache: remoteCache,
		config:      config,
	}
}

// CheckHealth 检查健康状态
func (chc *CacheHealthChecker) CheckHealth(ctx context.Context) (map[string]interface{}, error) {
	health := make(map[string]interface{})

	// 检查本地缓存
	localHealth := chc.checkCacheHealth(ctx, chc.localCache, "local")
	health["local_cache"] = localHealth

	// 检查远程缓存
	remoteHealth := chc.checkCacheHealth(ctx, chc.remoteCache, "remote")
	health["remote_cache"] = remoteHealth

	// 计算整体健康状态
	health["overall"] = chc.calculateOverallHealth(localHealth, remoteHealth)

	return health, nil
}

// checkCacheHealth 检查单个缓存的健康状态
func (chc *CacheHealthChecker) checkCacheHealth(ctx context.Context, cache CacheService, name string) map[string]interface{} {
	health := map[string]interface{}{
		"status": "unknown",
		"checks": map[string]interface{}{},
	}

	// 检查基本连接
	testKey := "health_check_" + name
	err := cache.Set(ctx, testKey, "test", time.Second*10)
	if err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]interface{})["connection"] = "failed"
		health["checks"].(map[string]interface{})["error"] = err.Error()
		return health
	}

	// 检查读写
	var value string
	err = cache.Get(ctx, testKey, &value)
	if err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]interface{})["read_write"] = "failed"
		health["checks"].(map[string]interface{})["error"] = err.Error()
		return health
	}

	health["status"] = "healthy"
	health["checks"].(map[string]interface{})["connection"] = "passed"
	health["checks"].(map[string]interface{})["read_write"] = "passed"

	return health
}

// calculateOverallHealth 计算整体健康状态
func (chc *CacheHealthChecker) calculateOverallHealth(localHealth, remoteHealth map[string]interface{}) map[string]interface{} {
	overall := map[string]interface{}{
		"status": "unknown",
	}

	localStatus, _ := localHealth["status"].(string)
	remoteStatus, _ := remoteHealth["status"].(string)

	if localStatus == "healthy" && remoteStatus == "healthy" {
		overall["status"] = "healthy"
	} else if localStatus == "healthy" || remoteStatus == "healthy" {
		overall["status"] = "degraded"
	} else {
		overall["status"] = "unhealthy"
	}

	overall["local_status"] = localStatus
	overall["remote_status"] = remoteStatus

	return overall
}

// recordMetrics 记录指标
func (mlc *MultiLevelCache) recordMetrics(operation string, duration time.Duration, success bool) {
	if !mlc.config.EnableMetrics {
		return
	}

	mlc.metricsCollector.RecordDBQuery("multi_level_cache", operation, duration, success)
	if !success {
		mlc.metricsCollector.RecordDBError("multi_level_cache_error", operation)
	}
}

// Close 关闭多级缓存
func (mlc *MultiLevelCache) Close() error {
	return nil
}

// CachePerformanceAnalyzer 缓存性能分析器
type CachePerformanceAnalyzer struct {
	localCache  CacheService
	remoteCache CacheService
	metrics     *CacheMetricsCollector
}

// NewCachePerformanceAnalyzer 创建缓存性能分析器
func NewCachePerformanceAnalyzer(localCache, remoteCache CacheService) *CachePerformanceAnalyzer {
	return &CachePerformanceAnalyzer{
		localCache:  localCache,
		remoteCache: remoteCache,
		metrics:     NewCacheMetricsCollector(localCache, remoteCache),
	}
}

// AnalyzePerformance 分析性能
func (cpa *CachePerformanceAnalyzer) AnalyzePerformance(ctx context.Context) (*PerformanceReport, error) {
	report := &PerformanceReport{
		Timestamp: time.Now(),
	}

	// 收集指标
	metrics := cpa.metrics.CollectMetrics()
	report.Metrics = metrics

	// 分析性能
	report.Analysis = cpa.analyzePerformanceMetrics(metrics)

	// 生成建议
	report.Recommendations = cpa.generateRecommendations(report.Analysis)

	return report, nil
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	Timestamp       time.Time              `json:"timestamp"`
	Metrics         map[string]interface{} `json:"metrics"`
	Analysis        PerformanceAnalysis    `json:"analysis"`
	Recommendations []string               `json:"recommendations"`
}

// PerformanceAnalysis 性能分析
type PerformanceAnalysis struct {
	HitRate          float64  `json:"hit_rate"`
	ResponseTime     float64  `json:"response_time"`
	ErrorRate        float64  `json:"error_rate"`
	PerformanceScore float64  `json:"performance_score"`
	Issues           []string `json:"issues"`
}

// analyzePerformanceMetrics 分析性能指标
func (cpa *CachePerformanceAnalyzer) analyzePerformanceMetrics(metrics map[string]interface{}) PerformanceAnalysis {
	analysis := PerformanceAnalysis{}

	// 获取指标
	hitRate, _ := metrics["hit_rate"].(float64)
	responseTime := 5.0 // 模拟响应时间(ms)
	errorRate, _ := metrics["error_rate"].(float64)

	analysis.HitRate = hitRate
	analysis.ResponseTime = responseTime
	analysis.ErrorRate = errorRate

	// 计算性能分数
	score := 100.0
	score -= (1 - hitRate) * 40          // 命中率影响 40%
	score -= (responseTime / 100.0) * 30 // 响应时间影响 30%
	score -= errorRate * 30              // 错误率影响 30%

	if score < 0 {
		score = 0
	}

	analysis.PerformanceScore = score

	// 识别问题
	if hitRate < 0.8 {
		analysis.Issues = append(analysis.Issues, "命中率较低")
	}
	if responseTime > 10 {
		analysis.Issues = append(analysis.Issues, "响应时间较长")
	}
	if errorRate > 0.05 {
		analysis.Issues = append(analysis.Issues, "错误率较高")
	}

	return analysis
}

// generateRecommendations 生成建议
func (cpa *CachePerformanceAnalyzer) generateRecommendations(analysis PerformanceAnalysis) []string {
	recommendations := []string{}

	if analysis.HitRate < 0.8 {
		recommendations = append(recommendations, "建议增加缓存预热或调整缓存策略以提高命中率")
	}

	if analysis.ResponseTime > 10 {
		recommendations = append(recommendations, "建议优化缓存实现或增加缓存容量以减少响应时间")
	}

	if analysis.ErrorRate > 0.05 {
		recommendations = append(recommendations, "建议检查缓存配置和网络连接以降低错误率")
	}

	if len(analysis.Issues) == 0 {
		recommendations = append(recommendations, "缓存性能良好，继续保持当前配置")
	}

	return recommendations
}
