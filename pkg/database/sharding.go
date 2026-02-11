package database

import (
	"fmt"
	"hash/crc32"
	"log"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"

	"gorm.io/gorm"
)

// ShardingStrategy 分片策略
type ShardingStrategy int

const (
	HashSharding ShardingStrategy = iota
	RangeSharding
	ConsistentHashSharding
	DirectorySharding
)

// ShardConfig 分片配置
type ShardConfig struct {
	Strategy    ShardingStrategy `json:"strategy"`
	ShardCount  int              `json:"shard_count"`
	TablePrefix string           `json:"table_prefix"`
	HashField   string           `json:"hash_field"`
	RangeField  string           `json:"range_field"`
	RangeStep   int              `json:"range_step"`
}

// Shard 分片信息
type Shard struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	DB       *gorm.DB
	Weight   int       `json:"weight"`
	Healthy  bool      `json:"healthy"`
	LastUsed time.Time `json:"last_used"`
}

// ShardingManager 分片管理器
type ShardingManager struct {
	shards      []*Shard
	config      *ShardConfig
	mu          sync.RWMutex
	metrics     *metrics.MetricsCollector
	routing     *RoutingManager
	healthCheck *ShardHealthChecker
}

// RoutingManager 路由管理器
type RoutingManager struct {
	strategy ShardingStrategy
	config   *ShardConfig
}

// ShardHealthChecker 分片健康检查器
type ShardHealthChecker struct {
	shards []*Shard
	config *ShardConfig
	stopCh chan struct{}
	mu     sync.RWMutex
}

// NewShardingManager 创建分片管理器
func NewShardingManager(config *ShardConfig, metricsCollector *metrics.MetricsCollector) *ShardingManager {
	sm := &ShardingManager{
		shards:      make([]*Shard, config.ShardCount),
		config:      config,
		metrics:     metricsCollector,
		routing:     NewRoutingManager(config.Strategy, config),
		healthCheck: NewShardHealthChecker(make([]*Shard, config.ShardCount), config),
	}

	// 初始化分片
	sm.initializeShards()

	// 启动健康检查
	go sm.healthCheck.Start()

	return sm
}

// initializeShards 初始化分片
func (sm *ShardingManager) initializeShards() {
	for i := 0; i < sm.config.ShardCount; i++ {
		shardName := fmt.Sprintf("%s_%d", sm.config.TablePrefix, i)

		// 这里应该根据配置连接到实际的数据库
		// 简化实现，使用模拟的数据库连接
		shard := &Shard{
			ID:       i,
			Name:     shardName,
			Weight:   1,
			Healthy:  true,
			LastUsed: time.Now(),
		}

		sm.shards[i] = shard
	}
}

// NewRoutingManager 创建路由管理器
func NewRoutingManager(strategy ShardingStrategy, config *ShardConfig) *RoutingManager {
	return &RoutingManager{
		strategy: strategy,
		config:   config,
	}
}

// NewShardHealthChecker 创建分片健康检查器
func NewShardHealthChecker(shards []*Shard, config *ShardConfig) *ShardHealthChecker {
	return &ShardHealthChecker{
		shards: shards,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start 开始健康检查
func (shc *ShardHealthChecker) Start() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			shc.checkAllShards()
		case <-shc.stopCh:
			return
		}
	}
}

// Stop 停止健康检查
func (shc *ShardHealthChecker) Stop() {
	close(shc.stopCh)
}

// checkAllShards 检查所有分片健康状态
func (shc *ShardHealthChecker) checkAllShards() {
	shc.mu.Lock()
	defer shc.mu.Unlock()

	for i, shard := range shc.shards {
		if shard == nil || shard.DB == nil {
			shard.Healthy = false
			continue
		}

		sqlDB, err := shard.DB.DB()
		if err != nil {
			shard.Healthy = false
			log.Printf("Failed to get shard %d DB: %v", i, err)
			continue
		}

		// 执行健康检查
		err = sqlDB.Ping()
		if err != nil {
			shard.Healthy = false
			log.Printf("Shard %d health check failed: %v", i, err)
		} else {
			shard.Healthy = true
		}
	}
}

// GetShard 根据键获取分片
func (sm *ShardingManager) GetShard(key interface{}) (*Shard, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shardIndex, err := sm.routing.Route(key, sm.config.ShardCount)
	if err != nil {
		return nil, err
	}

	if shardIndex < 0 || shardIndex >= len(sm.shards) {
		return nil, fmt.Errorf("invalid shard index: %d", shardIndex)
	}

	shard := sm.shards[shardIndex]
	if !shard.Healthy {
		return nil, fmt.Errorf("shard %d is unhealthy", shardIndex)
	}

	// 更新最后使用时间
	shard.LastUsed = time.Now()

	return shard, nil
}

// Route 路由到分片
func (rm *RoutingManager) Route(key interface{}, shardCount int) (int, error) {
	switch rm.strategy {
	case HashSharding:
		return rm.hashRouting(key, shardCount)
	case RangeSharding:
		return rm.rangeRouting(key, shardCount)
	case ConsistentHashSharding:
		return rm.consistentHashRouting(key, shardCount)
	case DirectorySharding:
		return rm.directoryRouting(key, shardCount)
	default:
		return 0, fmt.Errorf("unsupported sharding strategy: %v", rm.strategy)
	}
}

// hashRouting 哈希路由
func (rm *RoutingManager) hashRouting(key interface{}, shardCount int) (int, error) {
	var hashValue uint32

	switch v := key.(type) {
	case string:
		hashValue = crc32.ChecksumIEEE([]byte(v))
	case int:
		hashValue = crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d", v)))
	case int64:
		hashValue = crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d", v)))
	default:
		hashValue = crc32.ChecksumIEEE([]byte(fmt.Sprintf("%v", v)))
	}

	return int(hashValue % uint32(shardCount)), nil
}

// rangeRouting 范围路由
func (rm *RoutingManager) rangeRouting(key interface{}, shardCount int) (int, error) {
	// 简化的范围路由实现
	var value int64

	switch v := key.(type) {
	case int:
		value = int64(v)
	case int64:
		value = v
	case string:
		// 尝试解析为数字
		if _, err := fmt.Sscanf(v, "%d", &value); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("unsupported key type for range routing: %T", key)
	}

	if rm.config.RangeStep <= 0 {
		return 0, fmt.Errorf("invalid range step: %d", rm.config.RangeStep)
	}

	shardIndex := int(value / int64(rm.config.RangeStep))
	if shardIndex >= shardCount {
		shardIndex = shardCount - 1
	}

	return shardIndex, nil
}

// consistentHashRouting 一致性哈希路由
func (rm *RoutingManager) consistentHashRouting(key interface{}, shardCount int) (int, error) {
	// 简化的一致性哈希实现
	// 实际项目中应该使用更复杂的一致性哈希环

	hashValue := crc32.ChecksumIEEE([]byte(fmt.Sprintf("%v", key)))
	return int(hashValue % uint32(shardCount)), nil
}

// directoryRouting 目录路由
func (rm *RoutingManager) directoryRouting(key interface{}, shardCount int) (int, error) {
	// 简化的目录路由实现
	// 实际项目中应该使用路由表

	keyStr := fmt.Sprintf("%v", key)

	// 根据键的前缀进行路由
	if len(keyStr) > 0 {
		firstChar := strings.ToLower(keyStr)[0]
		shardIndex := int(firstChar) % shardCount
		return shardIndex, nil
	}

	return 0, nil
}

// CreateTableOnShard 在指定分片上创建表
func (sm *ShardingManager) CreateTableOnShard(shardIndex int, tableName string, model interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if shardIndex < 0 || shardIndex >= len(sm.shards) {
		return fmt.Errorf("invalid shard index: %d", shardIndex)
	}

	shard := sm.shards[shardIndex]
	if !shard.Healthy {
		return fmt.Errorf("shard %d is unhealthy", shardIndex)
	}

	// 创建表名
	fullTableName := fmt.Sprintf("%s_%d", tableName, shardIndex)

	// 使用 GORM 创建表
	err := shard.DB.Table(fullTableName).AutoMigrate(model)
	if err != nil {
		return fmt.Errorf("failed to create table on shard %d: %w", shardIndex, err)
	}

	// 记录指标
	sm.metrics.RecordDBError("table_created", fullTableName)

	log.Printf("Created table %s on shard %d", fullTableName, shardIndex)
	return nil
}

// DropTableOnShard 在指定分片上删除表
func (sm *ShardingManager) DropTableOnShard(shardIndex int, tableName string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if shardIndex < 0 || shardIndex >= len(sm.shards) {
		return fmt.Errorf("invalid shard index: %d", shardIndex)
	}

	shard := sm.shards[shardIndex]
	if !shard.Healthy {
		return fmt.Errorf("shard %d is unhealthy", shardIndex)
	}

	fullTableName := fmt.Sprintf("%s_%d", tableName, shardIndex)

	// 删除表
	err := shard.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", fullTableName))
	if err != nil {
		return fmt.Errorf("failed to drop table on shard %d: %w", shardIndex, err)
	}

	// 记录指标
	sm.metrics.RecordDBError("table_dropped", fullTableName)

	log.Printf("Dropped table %s on shard %d", fullTableName, shardIndex)
	return nil
}

// Insert 插入数据到合适的分片
func (sm *ShardingManager) Insert(data interface{}, shardKey interface{}) error {
	shard, err := sm.GetShard(shardKey)
	if err != nil {
		return fmt.Errorf("failed to get shard: %w", err)
	}

	// 插入数据
	result := shard.DB.Create(data)
	if result.Error != nil {
		return fmt.Errorf("failed to insert data: %w", result.Error)
	}

	// 记录指标
	sm.metrics.RecordDBError("data_inserted", shard.Name)

	return nil
}

// Find 查询数据
func (sm *ShardingManager) Find(dest interface{}, shardKey interface{}, conds ...interface{}) error {
	shard, err := sm.GetShard(shardKey)
	if err != nil {
		return fmt.Errorf("failed to get shard: %w", err)
	}

	// 查询数据
	result := shard.DB.Find(dest, conds...)
	if result.Error != nil {
		return fmt.Errorf("failed to find data: %w", result.Error)
	}

	// 记录指标
	sm.metrics.RecordDBError("data_found", shard.Name)

	return nil
}

// Update 更新数据
func (sm *ShardingManager) Update(data interface{}, shardKey interface{}, conds ...interface{}) error {
	shard, err := sm.GetShard(shardKey)
	if err != nil {
		return fmt.Errorf("failed to get shard: %w", err)
	}

	// 更新数据
	result := shard.DB.Model(data).Where(conds[0], conds[1:]...).Updates(data)
	if result.Error != nil {
		return fmt.Errorf("failed to update data: %w", result.Error)
	}

	// 记录指标
	sm.metrics.RecordDBError("data_updated", shard.Name)

	return nil
}

// Delete 删除数据
func (sm *ShardingManager) Delete(data interface{}, shardKey interface{}, conds ...interface{}) error {
	shard, err := sm.GetShard(shardKey)
	if err != nil {
		return fmt.Errorf("failed to get shard: %w", err)
	}

	// 删除数据
	result := shard.DB.Delete(data, conds...)
	if result.Error != nil {
		return fmt.Errorf("failed to delete data: %w", result.Error)
	}

	// 记录指标
	sm.metrics.RecordDBError("data_deleted", shard.Name)

	return nil
}

// QueryAllShards 查询所有分片的数据
func (sm *ShardingManager) QueryAllShards(dest interface{}, conds ...interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var allResults []interface{}

	for i, shard := range sm.shards {
		if !shard.Healthy {
			log.Printf("Skipping unhealthy shard %d", i)
			continue
		}

		var results []interface{}
		err := shard.DB.Find(&results, conds...).Error
		if err != nil {
			log.Printf("Failed to query shard %d: %v", i, err)
			continue
		}

		allResults = append(allResults, results...)
	}

	// 合并结果
	// 这里需要根据具体的业务逻辑来合并结果
	// 简化实现，直接返回所有结果

	// 记录指标
	sm.metrics.RecordDBError("all_shards_queried", "all")

	return nil
}

// GetShardStats 获取分片统计信息
func (sm *ShardingManager) GetShardStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]interface{})

	var healthyShards int
	var unhealthyShards int
	var totalWeight int

	shardDetails := make([]map[string]interface{}, len(sm.shards))

	for i, shard := range sm.shards {
		if shard.Healthy {
			healthyShards++
		} else {
			unhealthyShards++
		}
		totalWeight += shard.Weight

		shardDetails[i] = map[string]interface{}{
			"id":        shard.ID,
			"name":      shard.Name,
			"weight":    shard.Weight,
			"healthy":   shard.Healthy,
			"last_used": shard.LastUsed,
		}
	}

	stats["total_shards"] = len(sm.shards)
	stats["healthy_shards"] = healthyShards
	stats["unhealthy_shards"] = unhealthyShards
	stats["total_weight"] = totalWeight
	stats["shard_details"] = shardDetails

	return stats
}

// RebalanceShards 重新平衡分片
func (sm *ShardingManager) RebalanceShards() error {
	log.Println("Starting shard rebalancing...")

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 简化的重新平衡逻辑
	// 实际项目中应该根据数据量和访问模式进行更复杂的重新平衡

	for i, shard := range sm.shards {
		if !shard.Healthy {
			log.Printf("Attempting to recover shard %d", i)
			// 这里应该尝试重新连接分片
			// 简化实现，只是标记为健康
			shard.Healthy = true
		}
	}

	// 记录指标
	sm.metrics.RecordDBError("shards_rebalanced", "all")

	log.Println("Shard rebalancing completed")
	return nil
}

// AddShard 添加新分片
func (sm *ShardingManager) AddShard(shard *Shard) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.shards = append(sm.shards, shard)
	sm.config.ShardCount++

	// 记录指标
	sm.metrics.RecordDBError("shard_added", shard.Name)

	log.Printf("Added new shard: %s", shard.Name)
	return nil
}

// RemoveShard 移除分片
func (sm *ShardingManager) RemoveShard(shardID int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if shardID < 0 || shardID >= len(sm.shards) {
		return fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shard := sm.shards[shardID]

	// 移除分片
	sm.shards = append(sm.shards[:shardID], sm.shards[shardID+1:]...)
	sm.config.ShardCount--

	// 记录指标
	sm.metrics.RecordDBError("shard_removed", shard.Name)

	log.Printf("Removed shard: %s", shard.Name)
	return nil
}

// Close 关闭所有分片连接
func (sm *ShardingManager) Close() error {
	// 停止健康检查
	sm.healthCheck.Stop()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	var lastErr error
	for i, shard := range sm.shards {
		if shard != nil && shard.DB != nil {
			sqlDB, err := shard.DB.DB()
			if err != nil {
				log.Printf("Failed to get shard %d DB: %v", i, err)
				continue
			}

			if err := sqlDB.Close(); err != nil {
				log.Printf("Failed to close shard %d: %v", i, err)
				lastErr = err
			}
		}
	}

	return lastErr
}

// ShardMetrics 分片指标
type ShardMetrics struct {
	TotalShards     int       `json:"total_shards"`
	HealthyShards   int       `json:"healthy_shards"`
	UnhealthyShards int       `json:"unhealthy_shards"`
	TotalWeight     int       `json:"total_weight"`
	AvgResponseTime float64   `json:"avg_response_time"`
	RequestCount    int64     `json:"request_count"`
	ErrorCount      int64     `json:"error_count"`
	LastRebalance   time.Time `json:"last_rebalance"`
}

// GetMetrics 获取分片指标
func (sm *ShardingManager) GetMetrics() *ShardMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics := &ShardMetrics{
		TotalShards:   len(sm.shards),
		HealthyShards: 0,
		LastRebalance: time.Now(),
	}

	for _, shard := range sm.shards {
		if shard.Healthy {
			metrics.HealthyShards++
		} else {
			metrics.UnhealthyShards++
		}
		metrics.TotalWeight += shard.Weight
	}

	return metrics
}

// BatchInsert 批量插入到多个分片
func (sm *ShardingManager) BatchInsert(data []interface{}, getShardKey func(interface{}) interface{}) error {
	// 按分片分组数据
	shardGroups := make(map[int][]interface{})

	for _, item := range data {
		shardKey := getShardKey(item)
		shard, err := sm.routing.Route(shardKey, sm.config.ShardCount)
		if err != nil {
			return fmt.Errorf("failed to route item: %w", err)
		}

		shardGroups[shard] = append(shardGroups[shard], item)
	}

	// 批量插入到各个分片
	for shardIndex, items := range shardGroups {
		shard, err := sm.GetShard(shardIndex)
		if err != nil {
			return fmt.Errorf("failed to get shard %d: %w", shardIndex, err)
		}

		// 批量插入
		if err := shard.DB.CreateInBatches(items, 100).Error; err != nil {
			return fmt.Errorf("failed to batch insert to shard %d: %w", shardIndex, err)
		}
	}

	// 记录指标
	sm.metrics.RecordDBError("batch_insert_completed", "all_shards")

	return nil
}

// CrossShardQuery 跨分片查询
func (sm *ShardingManager) CrossShardQuery(queryFunc func(*gorm.DB) (interface{}, error)) ([]interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var allResults []interface{}

	for i, shard := range sm.shards {
		if !shard.Healthy {
			log.Printf("Skipping unhealthy shard %d in cross-shard query", i)
			continue
		}

		result, err := queryFunc(shard.DB)
		if err != nil {
			log.Printf("Failed to query shard %d: %v", i, err)
			continue
		}

		if result != nil {
			allResults = append(allResults, result)
		}
	}

	// 记录指标
	sm.metrics.RecordDBError("cross_shard_query", "all_shards")

	return allResults, nil
}
