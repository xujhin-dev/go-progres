package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"
)

// CacheWarmupManager 缓存预热管理器
type CacheWarmupManager struct {
	cache            CacheService
	metricsCollector *metrics.MetricsCollector
	strategies       map[string]WarmupStrategy
	config           *WarmupConfig
	scheduler        *WarmupScheduler
	loader           *DataLoader
}

// WarmupConfig 预热配置
type WarmupConfig struct {
	EnableScheduler   bool          `json:"enable_scheduler"`
	SchedulerInterval time.Duration `json:"scheduler_interval"`
	MaxConcurrency    int           `json:"max_concurrency"`
	EnableMetrics     bool          `json:"enable_metrics"`
	EnableRetry       bool          `json:"enable_retry"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	EnableProgress    bool          `json:"enable_progress"`
}

// WarmupStrategy 预热策略接口
type WarmupStrategy interface {
	Warmup(ctx context.Context, keys []string) (*WarmupResult, error)
	GetName() string
	GetPriority() int
	GetEstimatedTime() time.Duration
}

// WarmupResult 预热结果
type WarmupResult struct {
	Strategy    string                 `json:"strategy"`
	TotalKeys   int                    `json:"total_keys"`
	SuccessKeys int                    `json:"success_keys"`
	FailedKeys  int                    `json:"failed_keys"`
	Duration    time.Duration          `json:"duration"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Errors      []string               `json:"errors"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// WarmupScheduler 预热调度器
type WarmupScheduler struct {
	manager *CacheWarmupManager
	tasks   []WarmupTask
	mu      sync.RWMutex
	stopCh  chan struct{}
	config  *WarmupConfig
}

// WarmupTask 预热任务
type WarmupTask struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Strategy string                 `json:"strategy"`
	Keys     []string               `json:"keys"`
	Schedule string                 `json:"schedule"`
	Priority int                    `json:"priority"`
	Enabled  bool                   `json:"enabled"`
	LastRun  time.Time              `json:"last_run"`
	NextRun  time.Time              `json:"next_run"`
	Metadata map[string]interface{} `json:"metadata"`
}

// DataLoader 数据加载器
type DataLoader struct {
	cache            CacheService
	metricsCollector *metrics.MetricsCollector
	loaders          map[string]DataLoaderFunc
	mu               sync.RWMutex
}

// DataLoaderFunc 数据加载函数
type DataLoaderFunc func(ctx context.Context, key string) (interface{}, error)

// NewCacheWarmupManager 创建缓存预热管理器
func NewCacheWarmupManager(cache CacheService, metricsCollector *metrics.MetricsCollector, config *WarmupConfig) *CacheWarmupManager {
	cwm := &CacheWarmupManager{
		cache:            cache,
		metricsCollector: metricsCollector,
		strategies:       make(map[string]WarmupStrategy),
		config:           config,
		scheduler:        NewWarmupScheduler(config),
		loader:           NewDataLoader(cache, metricsCollector),
	}

	// 注册默认策略
	cwm.registerDefaultStrategies()

	// 启动调度器
	if config.EnableScheduler {
		go cwm.scheduler.Start()
	}

	return cwm
}

// NewWarmupScheduler 创建预热调度器
func NewWarmupScheduler(config *WarmupConfig) *WarmupScheduler {
	return &WarmupScheduler{
		tasks:  make([]WarmupTask, 0),
		stopCh: make(chan struct{}),
		config: config,
	}
}

// NewDataLoader 创建数据加载器
func NewDataLoader(cache CacheService, metricsCollector *metrics.MetricsCollector) *DataLoader {
	return &DataLoader{
		cache:            cache,
		metricsCollector: metricsCollector,
		loaders:          make(map[string]DataLoaderFunc),
	}
}

// registerDefaultStrategies 注册默认策略
func (cwm *CacheWarmupManager) registerDefaultStrategies() {
	// 立即预热策略
	cwm.strategies["immediate"] = &ImmediateWarmupStrategy{
		cache:  cwm.cache,
		loader: cwm.loader,
	}

	// 批量预热策略
	cwm.strategies["batch"] = &BatchWarmupStrategy{
		cache:     cwm.cache,
		loader:    cwm.loader,
		batchSize: 100,
	}

	// 渐进式预热策略
	cwm.strategies["progressive"] = &ProgressiveWarmupStrategy{
		cache:  cwm.cache,
		loader: cwm.loader,
		levels: []int{10, 50, 100, 500, 1000},
	}

	// 优先级预热策略
	cwm.strategies["priority"] = &PriorityWarmupStrategy{
		cache:    cwm.cache,
		loader:   cwm.loader,
		priority: map[string]int{},
	}

	// 智能预热策略
	cwm.strategies["smart"] = &SmartWarmupStrategy{
		cache:    cwm.cache,
		loader:   cwm.loader,
		analyzer: NewWarmupAnalyzer(),
	}
}

// Warmup 执行预热
func (cwm *CacheWarmupManager) Warmup(ctx context.Context, strategyName string, keys []string) (*WarmupResult, error) {
	strategy, exists := cwm.strategies[strategyName]
	if !exists {
		return nil, fmt.Errorf("warmup strategy %s not found", strategyName)
	}

	start := time.Now()
	defer func() {
		cwm.recordMetrics("warmup", time.Since(start), true)
	}()

	result, err := strategy.Warmup(ctx, keys)
	if err != nil {
		cwm.recordMetrics("warmup_error", time.Since(start), false)
		return nil, fmt.Errorf("failed to warmup keys with strategy %s: %w", strategyName, err)
	}

	// 记录预热指标
	cwm.recordWarmupMetrics(result)

	return result, nil
}

// AddTask 添加预热任务
func (cwm *CacheWarmupManager) AddTask(task WarmupTask) error {
	return cwm.scheduler.AddTask(task)
}

// RemoveTask 移除预热任务
func (cwm *CacheWarmupManager) RemoveTask(taskID string) error {
	return cwm.scheduler.RemoveTask(taskID)
}

// GetTasks 获取所有任务
func (cwm *CacheWarmupManager) GetTasks() []WarmupTask {
	return cwm.scheduler.GetTasks()
}

// RegisterLoader 注册数据加载器
func (cwm *CacheWarmupManager) RegisterLoader(name string, loader DataLoaderFunc) {
	cwm.loader.RegisterLoader(name, loader)
}

// recordMetrics 记录指标
func (cwm *CacheWarmupManager) recordMetrics(operation string, duration time.Duration, success bool) {
	if !cwm.config.EnableMetrics {
		return
	}

	cwm.metricsCollector.RecordDBQuery("cache_warmup", operation, duration, success)
	if !success {
		cwm.metricsCollector.RecordDBError("cache_warmup_error", operation)
	}
}

// recordWarmupMetrics 记录预热指标
func (cwm *CacheWarmupManager) recordWarmupMetrics(result *WarmupResult) {
	if !cwm.config.EnableMetrics {
		return
	}

	// 记录成功和失败的键数
	cwm.metricsCollector.RecordDBError("warmup_success_keys", fmt.Sprintf("%d", result.SuccessKeys))
	cwm.metricsCollector.RecordDBError("warmup_failed_keys", fmt.Sprintf("%d", result.FailedKeys))

	// 记录预热时间
	cwm.metricsCollector.RecordDBQuery("warmup_duration", result.Strategy, result.Duration, true)
}

// Start 启动调度器
func (ws *WarmupScheduler) Start() {
	ticker := time.NewTicker(ws.config.SchedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ws.runScheduledTasks()
		case <-ws.stopCh:
			return
		}
	}
}

// Stop 停止调度器
func (ws *WarmupScheduler) Stop() {
	close(ws.stopCh)
}

// AddTask 添加任务
func (ws *WarmupScheduler) AddTask(task WarmupTask) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// 设置下次运行时间
	if task.Schedule == "" {
		task.NextRun = time.Now()
	} else {
		// 简化的调度时间解析
		// 实际项目中应该使用更复杂的调度逻辑
		task.NextRun = time.Now().Add(time.Hour)
	}

	ws.tasks = append(ws.tasks, task)

	// 按优先级排序
	sort.Slice(ws.tasks, func(i, j int) bool {
		return ws.tasks[i].Priority > ws.tasks[j].Priority
	})

	return nil
}

// RemoveTask 移除任务
func (ws *WarmupScheduler) RemoveTask(taskID string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for i, task := range ws.tasks {
		if task.ID == taskID {
			ws.tasks = append(ws.tasks[:i], ws.tasks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("task %s not found", taskID)
}

// GetTasks 获取所有任务
func (ws *WarmupScheduler) GetTasks() []WarmupTask {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	tasks := make([]WarmupTask, len(ws.tasks))
	copy(tasks, ws.tasks)
	return tasks
}

// runScheduledTasks 运行调度任务
func (ws *WarmupScheduler) runScheduledTasks() {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	now := time.Now()

	for _, task := range ws.tasks {
		if !task.Enabled {
			continue
		}

		if now.After(task.NextRun) {
			// 运行任务
			go ws.runTask(task)

			// 更新下次运行时间
			task.LastRun = now
			task.NextRun = now.Add(time.Hour) // 简化实现
		}
	}
}

// runTask 运行任务
func (ws *WarmupScheduler) runTask(task WarmupTask) {
	log.Printf("Running warmup task: %s", task.Name)

	// 这里应该调用实际的预热逻辑
	// 简化实现，只是记录日志
	log.Printf("Task %s completed for %d keys", task.Name, len(task.Keys))
}

// RegisterLoader 注册数据加载器
func (dl *DataLoader) RegisterLoader(name string, loader DataLoaderFunc) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.loaders[name] = loader
}

// LoadData 加载数据
func (dl *DataLoader) LoadData(ctx context.Context, key string) (interface{}, error) {
	// 简化的数据加载逻辑
	// 实际项目中应该根据键名选择合适的加载器

	// 尝试从缓存获取
	var cached string
	err := dl.cache.Get(ctx, key, &cached)
	if err == nil && cached != "" {
		var data interface{}
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
	}

	// 模拟数据加载
	data := fmt.Sprintf("data_for_%s", key)

	// 缓存数据
	jsonData, _ := json.Marshal(data)
	dl.cache.Set(ctx, key, jsonData, time.Hour)

	return data, nil
}

// ImmediateWarmupStrategy 立即预热策略
type ImmediateWarmupStrategy struct {
	cache  CacheService
	loader *DataLoader
}

func (iws *ImmediateWarmupStrategy) Warmup(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		Strategy:  "immediate",
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	for _, key := range keys {
		data, err := iws.loader.LoadData(ctx, key)
		if err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load key %s: %v", key, err))
			continue
		}

		// 缓存数据
		jsonData, err := json.Marshal(data)
		if err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to marshal data for key %s: %v", key, err))
			continue
		}

		if err := iws.cache.Set(ctx, key, jsonData, time.Hour); err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to cache key %s: %v", key, err))
			continue
		}

		result.SuccessKeys++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (iws *ImmediateWarmupStrategy) GetName() string {
	return "immediate"
}

func (iws *ImmediateWarmupStrategy) GetPriority() int {
	return 100
}

func (iws *ImmediateWarmupStrategy) GetEstimatedTime() time.Duration {
	return time.Second * 10
}

// BatchWarmupStrategy 批量预热策略
type BatchWarmupStrategy struct {
	cache     CacheService
	loader    *DataLoader
	batchSize int
}

func (bws *BatchWarmupStrategy) Warmup(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		Strategy:  "batch",
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 分批处理
	for i := 0; i < len(keys); i += bws.batchSize {
		end := i + bws.batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		batchResult, err := bws.warmupBatch(ctx, batch)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Batch %d-%d failed: %v", i, end, err))
			continue
		}

		result.SuccessKeys += batchResult.SuccessKeys
		result.FailedKeys += batchResult.FailedKeys
		result.Errors = append(result.Errors, batchResult.Errors...)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (bws *BatchWarmupStrategy) warmupBatch(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
	}

	for _, key := range keys {
		_, err := bws.loader.LoadData(ctx, key)
		if err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load key %s: %v", key, err))
			continue
		}

		result.SuccessKeys++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (bws *BatchWarmupStrategy) GetName() string {
	return "batch"
}

func (bws *BatchWarmupStrategy) GetPriority() int {
	return 80
}

func (bws *BatchWarmupStrategy) GetEstimatedTime() time.Duration {
	return time.Second * 5
}

// ProgressiveWarmupStrategy 渐进式预热策略
type ProgressiveWarmupStrategy struct {
	cache  CacheService
	loader *DataLoader
	levels []int
}

func (pws *ProgressiveWarmupStrategy) Warmup(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		Strategy:  "progressive",
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 按级别渐进预热
	for _, level := range pws.levels {
		if level > len(keys) {
			level = len(keys)
		}

		levelKeys := keys[:level]
		_, err := pws.warmupLevel(ctx, levelKeys)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Level %d failed: %v", level, err))
			continue
		}

		// 等待一段时间再进行下一级别
		time.Sleep(time.Second * 2)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (pws *ProgressiveWarmupStrategy) warmupLevel(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
	}

	for _, key := range keys {
		_, err := pws.loader.LoadData(ctx, key)
		if err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load key %s: %v", key, err))
			continue
		}

		result.SuccessKeys++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (pws *ProgressiveWarmupStrategy) GetName() string {
	return "progressive"
}

func (pws *ProgressiveWarmupStrategy) GetPriority() int {
	return 70
}

func (pws *ProgressiveWarmupStrategy) GetEstimatedTime() time.Duration {
	return time.Second * 15
}

// PriorityWarmupStrategy 优先级预热策略
type PriorityWarmupStrategy struct {
	cache    CacheService
	loader   *DataLoader
	priority map[string]int
}

func (pws *PriorityWarmupStrategy) Warmup(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		Strategy:  "priority",
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 按优先级排序键
	sortedKeys := make([]string, len(keys))
	copy(sortedKeys, keys)

	sort.Slice(sortedKeys, func(i, j int) bool {
		priorityI := pws.priority[sortedKeys[i]]
		priorityJ := pws.priority[sortedKeys[j]]
		return priorityI > priorityJ
	})

	for _, key := range sortedKeys {
		_, err := pws.loader.LoadData(ctx, key)
		if err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load key %s: %v", key, err))
			continue
		}

		result.SuccessKeys++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (pws *PriorityWarmupStrategy) SetPriority(key string, priority int) {
	pws.priority[key] = priority
}

func (pws *PriorityWarmupStrategy) GetName() string {
	return "priority"
}

func (pws *PriorityWarmupStrategy) GetPriority() int {
	return 90
}

func (pws *PriorityWarmupStrategy) GetEstimatedTime() time.Duration {
	return time.Second * 8
}

// SmartWarmupStrategy 智能预热策略
type SmartWarmupStrategy struct {
	cache    CacheService
	loader   *DataLoader
	analyzer *WarmupAnalyzer
}

func (sws *SmartWarmupStrategy) Warmup(ctx context.Context, keys []string) (*WarmupResult, error) {
	result := &WarmupResult{
		Strategy:  "smart",
		TotalKeys: len(keys),
		StartTime: time.Now(),
		Errors:    make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 分析键的访问模式
	analysis := sws.analyzer.AnalyzeKeys(keys)
	result.Metadata["analysis"] = analysis

	// 根据分析结果选择预热策略
	sortedKeys := sws.analyzer.SortKeysByPriority(keys, analysis)

	for _, key := range sortedKeys {
		_, err := sws.loader.LoadData(ctx, key)
		if err != nil {
			result.FailedKeys++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load key %s: %v", key, err))
			continue
		}

		result.SuccessKeys++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (sws *SmartWarmupStrategy) GetName() string {
	return "smart"
}

func (sws *SmartWarmupStrategy) GetPriority() int {
	return 95
}

func (sws *SmartWarmupStrategy) GetEstimatedTime() time.Duration {
	return time.Second * 12
}

// WarmupAnalyzer 预热分析器
type WarmupAnalyzer struct {
	accessPatterns map[string]AccessPattern
	mu             sync.RWMutex
}

// AccessPattern 访问模式
type AccessPattern struct {
	Key         string    `json:"key"`
	AccessCount int       `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	Frequency   float64   `json:"frequency"`
	Priority    int       `json:"priority"`
}

// NewWarmupAnalyzer 创建预热分析器
func NewWarmupAnalyzer() *WarmupAnalyzer {
	return &WarmupAnalyzer{
		accessPatterns: make(map[string]AccessPattern),
	}
}

// AnalyzeKeys 分析键
func (wa *WarmupAnalyzer) AnalyzeKeys(keys []string) map[string]interface{} {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	analysis := make(map[string]interface{})

	totalAccess := 0
	for _, key := range keys {
		if pattern, exists := wa.accessPatterns[key]; exists {
			totalAccess += pattern.AccessCount
		}
	}

	analysis["total_access"] = totalAccess
	analysis["avg_access"] = float64(totalAccess) / float64(len(keys))
	analysis["unique_keys"] = len(keys)

	return analysis
}

// SortKeysByPriority 按优先级排序键
func (wa *WarmupAnalyzer) SortKeysByPriority(keys []string, analysis map[string]interface{}) []string {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	sortedKeys := make([]string, len(keys))
	copy(sortedKeys, keys)

	sort.Slice(sortedKeys, func(i, j int) bool {
		patternI := wa.accessPatterns[sortedKeys[i]]
		patternJ := wa.accessPatterns[sortedKeys[j]]

		if patternI.AccessCount == 0 && patternJ.AccessCount == 0 {
			return sortedKeys[i] < sortedKeys[j]
		}

		return patternI.AccessCount > patternJ.AccessCount
	})

	return sortedKeys
}

// RecordAccess 记录访问
func (wa *WarmupAnalyzer) RecordAccess(key string) {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	pattern, exists := wa.accessPatterns[key]
	if !exists {
		pattern = AccessPattern{
			Key:         key,
			AccessCount: 0,
			LastAccess:  time.Now(),
			Frequency:   0,
			Priority:    0,
		}
	}

	pattern.AccessCount++
	pattern.LastAccess = time.Now()
	pattern.Frequency = float64(pattern.AccessCount) / time.Since(time.Now().Add(-time.Hour*24)).Hours()

	// 计算优先级
	if pattern.AccessCount > 100 {
		pattern.Priority = 100
	} else if pattern.AccessCount > 50 {
		pattern.Priority = 80
	} else if pattern.AccessCount > 10 {
		pattern.Priority = 60
	} else {
		pattern.Priority = 40
	}

	wa.accessPatterns[key] = pattern
}

// GetWarmupMetrics 获取预热指标
func (cwm *CacheWarmupManager) GetWarmupMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// 策略信息
	strategies := make(map[string]interface{})
	for name, strategy := range cwm.strategies {
		strategies[name] = map[string]interface{}{
			"name":           strategy.GetName(),
			"priority":       strategy.GetPriority(),
			"estimated_time": strategy.GetEstimatedTime(),
		}
	}
	metrics["strategies"] = strategies

	// 任务信息
	tasks := cwm.GetTasks()
	metrics["tasks"] = map[string]interface{}{
		"total_tasks": len(tasks),
		"enabled_tasks": func() int {
			count := 0
			for _, task := range tasks {
				if task.Enabled {
					count++
				}
			}
			return count
		}(),
	}

	// 调度器信息
	metrics["scheduler"] = map[string]interface{}{
		"enabled":  cwm.config.EnableScheduler,
		"interval": cwm.config.SchedulerInterval,
	}

	return metrics, nil
}

// Close 关闭预热管理器
func (cwm *CacheWarmupManager) Close() error {
	if cwm.config.EnableScheduler {
		cwm.scheduler.Stop()
	}
	return nil
}
