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

// CacheConsistencyManager 缓存一致性管理器
type CacheConsistencyManager struct {
	cache            CacheService
	metricsCollector *metrics.MetricsCollector
	strategies       map[string]InvalidationStrategy
	eventBus         *EventBus
	config           *ConsistencyConfig
}

// ConsistencyConfig 一致性配置
type ConsistencyConfig struct {
	EnableEventBus   bool          `json:"enable_event_bus"`
	EventBusSize     int           `json:"event_bus_size"`
	EnableVersioning bool          `json:"enable_versioning"`
	EnableLocking    bool          `json:"enable_locking"`
	LockTimeout      time.Duration `json:"lock_timeout"`
	MaxRetries       int           `json:"max_retries"`
	RetryDelay       time.Duration `json:"retry_delay"`
	EnableMetrics    bool          `json:"enable_metrics"`
}

// InvalidationStrategy 失效策略接口
type InvalidationStrategy interface {
	Invalidate(ctx context.Context, keys []string) error
	GetName() string
	GetPriority() int
}

// EventBus 事件总线
type EventBus struct {
	subscribers map[string][]EventSubscriber
	mu          sync.RWMutex
	eventQueue  chan CacheEvent
	stopCh      chan struct{}
	config      *ConsistencyConfig
}

// CacheEvent 缓存事件
type CacheEvent struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// EventType 事件类型
type EventType string

const (
	EventSet     EventType = "set"
	EventDelete  EventType = "delete"
	EventExpire  EventType = "expire"
	EventRefresh EventType = "refresh"
	EventSync    EventType = "sync"
)

// EventSubscriber 事件订阅者
type EventSubscriber interface {
	Handle(ctx context.Context, event CacheEvent) error
	GetName() string
	GetEventTypes() []EventType
}

// NewCacheConsistencyManager 创建缓存一致性管理器
func NewCacheConsistencyManager(cache CacheService, metricsCollector *metrics.MetricsCollector, config *ConsistencyConfig) *CacheConsistencyManager {
	ccm := &CacheConsistencyManager{
		cache:            cache,
		metricsCollector: metricsCollector,
		strategies:       make(map[string]InvalidationStrategy),
		eventBus:         NewEventBus(config),
		config:           config,
	}

	// 注册默认策略
	ccm.registerDefaultStrategies()

	// 启动事件总线
	if config.EnableEventBus {
		go ccm.eventBus.Start()
	}

	return ccm
}

// NewEventBus 创建事件总线
func NewEventBus(config *ConsistencyConfig) *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventSubscriber),
		eventQueue:  make(chan CacheEvent, config.EventBusSize),
		stopCh:      make(chan struct{}),
		config:      config,
	}
}

// registerDefaultStrategies 注册默认策略
func (ccm *CacheConsistencyManager) registerDefaultStrategies() {
	// 立即失效策略
	ccm.strategies["immediate"] = &ImmediateInvalidationStrategy{
		cache: ccm.cache,
	}

	// 延迟失效策略
	ccm.strategies["delayed"] = &DelayedInvalidationStrategy{
		cache: ccm.cache,
		delay: time.Second * 5,
	}

	// 批量失效策略
	ccm.strategies["batch"] = &BatchInvalidationStrategy{
		cache:     ccm.cache,
		batchSize: 100,
		timeout:   time.Second * 10,
	}

	// 版本化失效策略
	ccm.strategies["versioned"] = &VersionedInvalidationStrategy{
		cache: ccm.cache,
	}

	// 依赖失效策略
	ccm.strategies["dependency"] = &DependencyInvalidationStrategy{
		cache:        ccm.cache,
		dependencies: make(map[string][]string),
	}
}

// Invalidate 失效缓存
func (ccm *CacheConsistencyManager) Invalidate(ctx context.Context, strategyName string, keys []string) error {
	strategy, exists := ccm.strategies[strategyName]
	if !exists {
		return fmt.Errorf("invalidation strategy %s not found", strategyName)
	}

	start := time.Now()
	defer func() {
		ccm.recordMetrics("invalidate", time.Since(start), true)
	}()

	err := strategy.Invalidate(ctx, keys)
	if err != nil {
		ccm.recordMetrics("invalidate_error", time.Since(start), false)
		return fmt.Errorf("failed to invalidate keys with strategy %s: %w", strategyName, err)
	}

	// 发送失效事件
	if ccm.config.EnableEventBus {
		for _, key := range keys {
			event := CacheEvent{
				ID:        generateEventID(),
				Type:      EventDelete,
				Key:       key,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"strategy": strategyName,
				},
			}
			ccm.eventBus.Publish(event)
		}
	}

	return nil
}

// Subscribe 订阅事件
func (ccm *CacheConsistencyManager) Subscribe(subscriber EventSubscriber) error {
	return ccm.eventBus.Subscribe(subscriber)
}

// Publish 发布事件
func (ccm *CacheConsistencyManager) Publish(event CacheEvent) {
	ccm.eventBus.Publish(event)
}

// GetStrategies 获取所有策略
func (ccm *CacheConsistencyManager) GetStrategies() map[string]InvalidationStrategy {
	return ccm.strategies
}

// recordMetrics 记录指标
func (ccm *CacheConsistencyManager) recordMetrics(operation string, duration time.Duration, success bool) {
	if !ccm.config.EnableMetrics {
		return
	}

	ccm.metricsCollector.RecordDBQuery("cache_consistency", operation, duration, success)
	if !success {
		ccm.metricsCollector.RecordDBError("cache_consistency_error", operation)
	}
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// Start 启动事件总线
func (eb *EventBus) Start() {
	for {
		select {
		case event := <-eb.eventQueue:
			eb.handleEvent(event)
		case <-eb.stopCh:
			return
		}
	}
}

// Stop 停止事件总线
func (eb *EventBus) Stop() {
	close(eb.stopCh)
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(subscriber EventSubscriber) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eventTypes := subscriber.GetEventTypes()
	for _, eventType := range eventTypes {
		eb.subscribers[string(eventType)] = append(eb.subscribers[string(eventType)], subscriber)
	}

	return nil
}

// Publish 发布事件
func (eb *EventBus) Publish(event CacheEvent) {
	select {
	case eb.eventQueue <- event:
	default:
		log.Printf("Event queue is full, dropping event: %s", event.ID)
	}
}

// handleEvent 处理事件
func (eb *EventBus) handleEvent(event CacheEvent) {
	eb.mu.RLock()
	subscribers, exists := eb.subscribers[string(event.Type)]
	eb.mu.RUnlock()

	if !exists {
		return
	}

	for _, subscriber := range subscribers {
		go func(s EventSubscriber) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			if err := s.Handle(ctx, event); err != nil {
				log.Printf("Event subscriber %s failed to handle event %s: %v", s.GetName(), event.ID, err)
			}
		}(subscriber)
	}
}

// ImmediateInvalidationStrategy 立即失效策略
type ImmediateInvalidationStrategy struct {
	cache CacheService
}

func (iis *ImmediateInvalidationStrategy) Invalidate(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := iis.cache.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}
	return nil
}

func (iis *ImmediateInvalidationStrategy) GetName() string {
	return "immediate"
}

func (iis *ImmediateInvalidationStrategy) GetPriority() int {
	return 100 // 最高优先级
}

// DelayedInvalidationStrategy 延迟失效策略
type DelayedInvalidationStrategy struct {
	cache  CacheService
	delay  time.Duration
	timers map[string]*time.Timer
	mu     sync.Mutex
}

func (dis *DelayedInvalidationStrategy) Invalidate(ctx context.Context, keys []string) error {
	dis.mu.Lock()
	defer dis.mu.Unlock()

	if dis.timers == nil {
		dis.timers = make(map[string]*time.Timer)
	}

	for _, key := range keys {
		// 取消之前的定时器
		if timer, exists := dis.timers[key]; exists {
			timer.Stop()
		}

		// 创建新的延迟失效定时器
		dis.timers[key] = time.AfterFunc(dis.delay, func() {
			dis.cache.Delete(context.Background(), key)
			dis.mu.Lock()
			delete(dis.timers, key)
			dis.mu.Unlock()
		})
	}

	return nil
}

func (dis *DelayedInvalidationStrategy) GetName() string {
	return "delayed"
}

func (dis *DelayedInvalidationStrategy) GetPriority() int {
	return 80
}

// BatchInvalidationStrategy 批量失效策略
type BatchInvalidationStrategy struct {
	cache     CacheService
	batchSize int
	timeout   time.Duration
	queue     []string
	mu        sync.Mutex
}

func (bis *BatchInvalidationStrategy) Invalidate(ctx context.Context, keys []string) error {
	bis.mu.Lock()
	defer bis.mu.Unlock()

	bis.queue = append(bis.queue, keys...)

	// 如果队列达到批量大小，执行批量删除
	if len(bis.queue) >= bis.batchSize {
		return bis.flushBatch(ctx)
	}

	return nil
}

func (bis *BatchInvalidationStrategy) flushBatch(ctx context.Context) error {
	if len(bis.queue) == 0 {
		return nil
	}

	keysToInvalidate := bis.queue
	bis.queue = nil

	// 执行批量删除
	for _, key := range keysToInvalidate {
		if err := bis.cache.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}

	return nil
}

func (bis *BatchInvalidationStrategy) GetName() string {
	return "batch"
}

func (bis *BatchInvalidationStrategy) GetPriority() int {
	return 60
}

// VersionedInvalidationStrategy 版本化失效策略
type VersionedInvalidationStrategy struct {
	cache CacheService
}

func (vis *VersionedInvalidationStrategy) Invalidate(ctx context.Context, keys []string) error {
	for _, key := range keys {
		// 增加版本号
		versionKey := fmt.Sprintf("%s:version", key)
		newVersion := time.Now().UnixNano()

		if err := vis.cache.Set(ctx, versionKey, newVersion, 0); err != nil {
			return fmt.Errorf("failed to update version for key %s: %w", key, err)
		}
	}
	return nil
}

func (vis *VersionedInvalidationStrategy) GetName() string {
	return "versioned"
}

func (vis *VersionedInvalidationStrategy) GetPriority() int {
	return 70
}

// DependencyInvalidationStrategy 依赖失效策略
type DependencyInvalidationStrategy struct {
	cache        CacheService
	dependencies map[string][]string
	mu           sync.RWMutex
}

func (dis *DependencyInvalidationStrategy) Invalidate(ctx context.Context, keys []string) error {
	dis.mu.Lock()
	defer dis.mu.Unlock()

	// 收集所有依赖的键
	allKeys := make(map[string]bool)
	for _, key := range keys {
		allKeys[key] = true

		// 添加依赖的键
		if deps, exists := dis.dependencies[key]; exists {
			for _, dep := range deps {
				allKeys[dep] = true
			}
		}
	}

	// 失效所有键
	for key := range allKeys {
		if err := dis.cache.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}

	return nil
}

func (dis *DependencyInvalidationStrategy) AddDependency(key, dependent string) {
	dis.mu.Lock()
	defer dis.mu.Unlock()

	if dis.dependencies[key] == nil {
		dis.dependencies[key] = make([]string, 0)
	}
	dis.dependencies[key] = append(dis.dependencies[key], dependent)
}

func (dis *DependencyInvalidationStrategy) GetName() string {
	return "dependency"
}

func (dis *DependencyInvalidationStrategy) GetPriority() int {
	return 90
}

// CacheVersioning 缓存版本控制
type CacheVersioning struct {
	cache    CacheService
	versions map[string]int64
	mu       sync.RWMutex
}

// NewCacheVersioning 创建缓存版本控制
func NewCacheVersioning(cache CacheService) *CacheVersioning {
	return &CacheVersioning{
		cache:    cache,
		versions: make(map[string]int64),
	}
}

// GetVersion 获取版本
func (cv *CacheVersioning) GetVersion(ctx context.Context, key string) (int64, error) {
	cv.mu.RLock()
	version, exists := cv.versions[key]
	cv.mu.RUnlock()

	if !exists {
		// 从缓存获取版本
		versionKey := fmt.Sprintf("%s:version", key)
		var versionStr string
		err := cv.cache.Get(ctx, versionKey, &versionStr)
		if err != nil {
			return 0, err
		}

		if versionStr == "" {
			version = 0
		} else {
			var v int64
			if err := json.Unmarshal([]byte(versionStr), &v); err != nil {
				return 0, err
			}
			version = v
		}

		cv.mu.Lock()
		cv.versions[key] = version
		cv.mu.Unlock()
	}

	return version, nil
}

// SetVersion 设置版本
func (cv *CacheVersioning) SetVersion(ctx context.Context, key string, version int64) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	cv.versions[key] = version

	versionKey := fmt.Sprintf("%s:version", key)
	versionData, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	return cv.cache.Set(ctx, versionKey, versionData, 0)
}

// IncrementVersion 增加版本
func (cv *CacheVersioning) IncrementVersion(ctx context.Context, key string) (int64, error) {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	version := cv.versions[key] + 1
	cv.versions[key] = version

	return version, cv.SetVersion(ctx, key, version)
}

// CacheLocking 缓存锁
type CacheLocking struct {
	cache CacheService
	mu    sync.Mutex
}

// NewCacheLocking 创建缓存锁
func NewCacheLocking(cache CacheService) *CacheLocking {
	return &CacheLocking{
		cache: cache,
	}
}

// Lock 获取锁
func (cl *CacheLocking) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)

	// 尝试设置锁
	err := cl.cache.Set(ctx, lockKey, "locked", ttl)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock for key %s: %w", key, err)
	}

	// 检查是否成功获取锁
	exists, err := cl.cache.Exists(ctx, lockKey)
	if err != nil {
		return false, fmt.Errorf("failed to check lock existence for key %s: %w", key, err)
	}

	return exists, nil
}

// Unlock 释放锁
func (cl *CacheLocking) Unlock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	return cl.cache.Delete(ctx, lockKey)
}

// TryLock 尝试获取锁
func (cl *CacheLocking) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return cl.Lock(ctx, key, ttl)
}

// CacheConsistencyChecker 一致性检查器
type CacheConsistencyChecker struct {
	cache            CacheService
	metricsCollector *metrics.MetricsCollector
	config           *ConsistencyConfig
}

// NewCacheConsistencyChecker 创建一致性检查器
func NewCacheConsistencyChecker(cache CacheService, metricsCollector *metrics.MetricsCollector, config *ConsistencyConfig) *CacheConsistencyChecker {
	return &CacheConsistencyChecker{
		cache:            cache,
		metricsCollector: metricsCollector,
		config:           config,
	}
}

// CheckConsistency 检查一致性
func (ccc *CacheConsistencyChecker) CheckConsistency(ctx context.Context, keys []string) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		CheckedKeys: len(keys),
		Timestamp:   time.Now(),
		Issues:      make([]ConsistencyIssue, 0),
	}

	for _, key := range keys {
		issues := ccc.checkKeyConsistency(ctx, key)
		report.Issues = append(report.Issues, issues...)
	}

	// 计算一致性分数
	report.ConsistencyScore = ccc.calculateConsistencyScore(report)

	return report, nil
}

// ConsistencyReport 一致性报告
type ConsistencyReport struct {
	CheckedKeys      int                `json:"checked_keys"`
	Timestamp        time.Time          `json:"timestamp"`
	Issues           []ConsistencyIssue `json:"issues"`
	ConsistencyScore float64            `json:"consistency_score"`
}

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	Key         string    `json:"key"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
}

// checkKeyConsistency 检查单个键的一致性
func (ccc *CacheConsistencyChecker) checkKeyConsistency(ctx context.Context, key string) []ConsistencyIssue {
	var issues []ConsistencyIssue

	// 检查键是否存在
	exists, err := ccc.cache.Exists(ctx, key)
	if err != nil {
		issues = append(issues, ConsistencyIssue{
			Key:         key,
			Type:        "existence_check",
			Description: fmt.Sprintf("Failed to check key existence: %v", err),
			Severity:    "error",
			Timestamp:   time.Now(),
		})
		return issues
	}

	if !exists {
		issues = append(issues, ConsistencyIssue{
			Key:         key,
			Type:        "missing_key",
			Description: "Key does not exist",
			Severity:    "warning",
			Timestamp:   time.Now(),
		})
	}

	// 检查 TTL
	ttl, err := ccc.cache.GetWithTTL(ctx, key, "")
	if err != nil {
		issues = append(issues, ConsistencyIssue{
			Key:         key,
			Type:        "ttl_check",
			Description: fmt.Sprintf("Failed to check TTL: %v", err),
			Severity:    "error",
			Timestamp:   time.Now(),
		})
	} else if ttl == 0 {
		issues = append(issues, ConsistencyIssue{
			Key:         key,
			Type:        "no_ttl",
			Description: "Key has no TTL set",
			Severity:    "info",
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// calculateConsistencyScore 计算一致性分数
func (ccc *CacheConsistencyChecker) calculateConsistencyScore(report *ConsistencyReport) float64 {
	if report.CheckedKeys == 0 {
		return 100.0
	}

	// 根据问题严重程度计算分数
	totalScore := 100.0
	for _, issue := range report.Issues {
		switch issue.Severity {
		case "error":
			totalScore -= 20.0
		case "warning":
			totalScore -= 10.0
		case "info":
			totalScore -= 5.0
		}
	}

	if totalScore < 0 {
		totalScore = 0
	}

	return totalScore
}

// CacheConsistencyMetrics 一致性指标
type CacheConsistencyMetrics struct {
	TotalInvalidations  int64         `json:"total_invalidations"`
	InvalidationRate    float64       `json:"invalidation_rate"`
	AvgInvalidationTime time.Duration `json:"avg_invalidation_time"`
	ConsistencyScore    float64       `json:"consistency_score"`
	EventBusSize        int           `json:"event_bus_size"`
	ActiveStrategies    int           `json:"active_strategies"`
}

// GetMetrics 获取一致性指标
func (ccm *CacheConsistencyManager) GetMetrics(ctx context.Context) (*CacheConsistencyMetrics, error) {
	metrics := &CacheConsistencyMetrics{
		ActiveStrategies: len(ccm.strategies),
	}

	// 获取事件总线大小
	if ccm.config.EnableEventBus {
		metrics.EventBusSize = len(ccm.eventBus.eventQueue)
	}

	// 检查一致性
	checker := NewCacheConsistencyChecker(ccm.cache, ccm.metricsCollector, ccm.config)

	// 获取一些示例键进行检查
	sampleKeys := []string{"user:1", "config:app", "cache:stats"}
	report, err := checker.CheckConsistency(ctx, sampleKeys)
	if err != nil {
		log.Printf("Failed to check consistency: %v", err)
	} else {
		metrics.ConsistencyScore = report.ConsistencyScore
	}

	return metrics, nil
}

// Close 关闭一致性管理器
func (ccm *CacheConsistencyManager) Close() error {
	if ccm.config.EnableEventBus {
		ccm.eventBus.Stop()
	}
	return nil
}
