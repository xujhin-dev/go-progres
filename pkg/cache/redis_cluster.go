package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"

	"github.com/go-redis/redis/v8"
)

// RedisCluster Redis 集群缓存
type RedisCluster struct {
	cluster          *redis.ClusterClient
	metricsCollector *metrics.MetricsCollector
	config           *RedisClusterConfig
	keyRouter        *KeyRouter
	healthChecker    *ClusterHealthChecker
}

// RedisClusterConfig Redis 集群配置
type RedisClusterConfig struct {
	Nodes               []string      `json:"nodes"`
	Password            string        `json:"password"`
	MaxRetries          int           `json:"max_retries"`
	PoolSize            int           `json:"pool_size"`
	MinIdleConns        int           `json:"min_idle_conns"`
	MaxIdleConns        int           `json:"max_idle_conns"`
	ConnMaxLifetime     time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime     time.Duration `json:"conn_max_idle_time"`
	EnablePipeline      bool          `json:"enable_pipeline"`
	EnableMetrics       bool          `json:"enable_metrics"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// KeyRouter 键路由器
type KeyRouter struct {
	clusterNodes []string
	mu           sync.RWMutex
}

// ClusterHealthChecker 集群健康检查器
type ClusterHealthChecker struct {
	cluster *redis.ClusterClient
	config  *RedisClusterConfig
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// NewRedisCluster 创建 Redis 集群
func NewRedisCluster(config *RedisClusterConfig, metricsCollector *metrics.MetricsCollector) (*RedisCluster, error) {
	// 创建 Redis 集群客户端
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      config.Nodes,
		Password:   config.Password,
		MaxRetries: config.MaxRetries,
		PoolSize:   config.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
	}

	redisCluster := &RedisCluster{
		cluster:          rdb,
		metricsCollector: metricsCollector,
		config:           config,
		keyRouter:        NewKeyRouter(config.Nodes),
		healthChecker:    NewClusterHealthChecker(rdb, config),
	}

	// 启动健康检查
	go redisCluster.healthChecker.Start()

	return redisCluster, nil
}

// NewKeyRouter 创建键路由器
func NewKeyRouter(nodes []string) *KeyRouter {
	return &KeyRouter{
		clusterNodes: nodes,
	}
}

// NewClusterHealthChecker 创建集群健康检查器
func NewClusterHealthChecker(cluster *redis.ClusterClient, config *RedisClusterConfig) *ClusterHealthChecker {
	return &ClusterHealthChecker{
		cluster: cluster,
		config:  config,
		stopCh:  make(chan struct{}),
	}
}

// Start 开始健康检查
func (chc *ClusterHealthChecker) Start() {
	ticker := time.NewTicker(chc.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			chc.checkClusterHealth()
		case <-chc.stopCh:
			return
		}
	}
}

// Stop 停止健康检查
func (chc *ClusterHealthChecker) Stop() {
	close(chc.stopCh)
}

// checkClusterHealth 检查集群健康状态
func (chc *ClusterHealthChecker) checkClusterHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	// 检查集群状态
	result := chc.cluster.ClusterInfo(ctx)
	if result.Err() != nil {
		log.Printf("Redis cluster health check failed: %v", result.Err())
		return
	}

	clusterInfo := result.Val()
	if !strings.Contains(clusterInfo, "cluster_state:ok") {
		log.Printf("Redis cluster state is not ok: %s", clusterInfo)
	}
}

// Get 获取缓存值
func (rc *RedisCluster) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		rc.recordMetrics("get", dur, true)
	}()

	result := rc.cluster.Get(ctx, key)
	if result.Err() == redis.Nil {
		rc.recordMetrics("get_miss", time.Since(start), true)
		return "", nil
	}

	if result.Err() != nil {
		rc.recordMetrics("get_error", time.Since(start), false)
		return "", fmt.Errorf("failed to get key %s: %w", key, result.Err())
	}

	rc.recordMetrics("get_hit", time.Since(start), true)
	return result.Val(), nil
}

// Set 设置缓存值
func (rc *RedisCluster) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		rc.recordMetrics("set", dur, true)
	}()

	result := rc.cluster.Set(ctx, key, value, expiration)
	if result.Err() != nil {
		rc.recordMetrics("set_error", time.Since(start), false)
		return fmt.Errorf("failed to set key %s: %w", key, result.Err())
	}

	return nil
}

// Delete 删除缓存值
func (rc *RedisCluster) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		rc.recordMetrics("delete", dur, true)
	}()

	result := rc.cluster.Del(ctx, key)
	if result.Err() != nil {
		rc.recordMetrics("delete_error", time.Since(start), false)
		return fmt.Errorf("failed to delete key %s: %w", key, result.Err())
	}

	return nil
}

// Exists 检查键是否存在
func (rc *RedisCluster) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("exists", duration, true)
	}()

	result := rc.cluster.Exists(ctx, key)
	if result.Err() != nil {
		rc.recordMetrics("exists_error", time.Since(start), false)
		return false, fmt.Errorf("failed to check key %s: %w", key, result.Err())
	}

	return result.Val() > 0, nil
}

// Expire 设置键的过期时间
func (rc *RedisCluster) Expire(ctx context.Context, key string, expiration time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("expire", duration, true)
	}()

	result := rc.cluster.Expire(ctx, key, expiration)
	if result.Err() != nil {
		rc.recordMetrics("expire_error", time.Since(start), false)
		return fmt.Errorf("failed to expire key %s: %w", key, result.Err())
	}

	return nil
}

// TTL 获取键的剩余过期时间
func (rc *RedisCluster) TTL(ctx context.Context, key string) (time.Duration, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("ttl", duration, true)
	}()

	result := rc.cluster.TTL(ctx, key)
	if result.Err() != nil {
		rc.recordMetrics("ttl_error", time.Since(start), false)
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, result.Err())
	}

	return result.Val(), nil
}

// Increment 原子递增
func (rc *RedisCluster) Increment(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("increment", duration, true)
	}()

	result := rc.cluster.Incr(ctx, key)
	if result.Err() != nil {
		rc.recordMetrics("increment_error", time.Since(start), false)
		return 0, fmt.Errorf("failed to increment key %s: %w", key, result.Err())
	}

	return result.Val(), nil
}

// Decrement 原子递减
func (rc *RedisCluster) Decrement(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("decrement", duration, true)
	}()

	result := rc.cluster.Decr(ctx, key)
	if result.Err() != nil {
		rc.recordMetrics("decrement_error", time.Since(start), false)
		return 0, fmt.Errorf("failed to decrement key %s: %w", key, result.Err())
	}

	return result.Val(), nil
}

// SetJSON 设置 JSON 值
func (rc *RedisCluster) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return rc.Set(ctx, key, jsonData, expiration)
}

// GetJSON 获取 JSON 值
func (rc *RedisCluster) GetJSON(ctx context.Context, key string, dest interface{}) error {
	value, err := rc.Get(ctx, key)
	if err != nil {
		return err
	}

	if value == "" {
		return nil
	}

	return json.Unmarshal([]byte(value), dest)
}

// MGet 批量获取
func (rc *RedisCluster) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("mget", duration, true)
	}()

	result := rc.cluster.MGet(ctx, keys...)
	if result.Err() != nil {
		rc.recordMetrics("mget_error", time.Since(start), false)
		return nil, fmt.Errorf("failed to MGet keys: %w", result.Err())
	}

	return result.Val(), nil
}

// MSet 批量设置
func (rc *RedisCluster) MSet(ctx context.Context, pairs ...interface{}) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("mset", duration, true)
	}()

	result := rc.cluster.MSet(ctx, pairs...)
	if result.Err() != nil {
		rc.recordMetrics("mset_error", time.Since(start), false)
		return fmt.Errorf("failed to MSet: %w", result.Err())
	}

	return nil
}

// Pipeline 批量操作
func (rc *RedisCluster) Pipeline(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.recordMetrics("pipeline", duration, true)
	}()

	_, err := rc.cluster.Pipeline().Exec(ctx)
	if err != nil {
		rc.recordMetrics("pipeline_error", time.Since(start), false)
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	return nil
}

// GetClusterInfo 获取集群信息
func (rc *RedisCluster) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	info := rc.cluster.ClusterInfo(ctx)
	if info.Err() != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", info.Err())
	}

	nodes := rc.cluster.ClusterNodes(ctx)
	if nodes.Err() != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", nodes.Err())
	}

	clusterInfo := map[string]interface{}{
		"cluster_info":  info.Val(),
		"cluster_nodes": nodes.Val(),
		"node_count":    len(strings.Split(nodes.Val(), "\n")),
	}

	return clusterInfo, nil
}

// GetStats 获取统计信息
func (rc *RedisCluster) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取集群信息
	clusterInfo, err := rc.GetClusterInfo(ctx)
	if err != nil {
		return nil, err
	}
	stats["cluster"] = clusterInfo

	// 获取每个节点的信息
	nodeStats := make([]map[string]interface{}, 0)
	for _, node := range rc.config.Nodes {
		nodeInfo := rc.getNodeInfo(ctx, node)
		nodeStats = append(nodeStats, nodeInfo)
	}
	stats["nodes"] = nodeStats

	// 获取连接池统计
	stats["pool_stats"] = rc.getPoolStats()

	return stats, nil
}

// getNodeInfo 获取节点信息
func (rc *RedisCluster) getNodeInfo(ctx context.Context, node string) map[string]interface{} {
	info := map[string]interface{}{
		"node":   node,
		"status": "unknown",
	}

	// 简化的节点信息获取
	// 实际项目中应该连接到具体节点获取详细信息
	return info
}

// getPoolStats 获取连接池统计
func (rc *RedisCluster) getPoolStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Redis 集群客户端的连接池统计
	// 这里简化实现，实际项目中应该获取详细的连接池信息

	stats["pool_size"] = rc.config.PoolSize
	stats["max_idle_conns"] = rc.config.MaxIdleConns
	stats["min_idle_conns"] = rc.config.MinIdleConns

	return stats
}

// recordMetrics 记录指标
func (rc *RedisCluster) recordMetrics(operation string, duration time.Duration, success bool) {
	if !rc.config.EnableMetrics {
		return
	}

	// 记录操作指标
	rc.metricsCollector.RecordDBQuery("redis_cluster", operation, duration, success)

	// 记录错误指标
	if !success {
		rc.metricsCollector.RecordDBError("redis_cluster_error", operation)
	}
}

// Close 关闭连接
func (rc *RedisCluster) Close() error {
	// 停止健康检查
	rc.healthChecker.Stop()

	// 关闭集群连接
	return rc.cluster.Close()
}

// RedisClusterManager Redis 集群管理器
type RedisClusterManager struct {
	clusters map[string]*RedisCluster
	mu       sync.RWMutex
	config   *RedisClusterConfig
	metrics  *metrics.MetricsCollector
}

// NewRedisClusterManager 创建 Redis 集群管理器
func NewRedisClusterManager(config *RedisClusterConfig, metricsCollector *metrics.MetricsCollector) *RedisClusterManager {
	return &RedisClusterManager{
		clusters: make(map[string]*RedisCluster),
		config:   config,
		metrics:  metricsCollector,
	}
}

// AddCluster 添加集群
func (rcm *RedisClusterManager) AddCluster(name string, nodes []string) error {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()

	// 创建集群配置
	clusterConfig := *rcm.config
	clusterConfig.Nodes = nodes

	// 创建集群
	cluster, err := NewRedisCluster(&clusterConfig, rcm.metrics)
	if err != nil {
		return fmt.Errorf("failed to create cluster %s: %w", name, err)
	}

	rcm.clusters[name] = cluster
	return nil
}

// GetCluster 获取集群
func (rcm *RedisClusterManager) GetCluster(name string) (*RedisCluster, error) {
	rcm.mu.RLock()
	defer rcm.mu.RUnlock()

	cluster, exists := rcm.clusters[name]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	return cluster, nil
}

// RemoveCluster 移除集群
func (rcm *RedisClusterManager) RemoveCluster(name string) error {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()

	cluster, exists := rcm.clusters[name]
	if !exists {
		return fmt.Errorf("cluster %s not found", name)
	}

	// 关闭集群连接
	if err := cluster.Close(); err != nil {
		return fmt.Errorf("failed to close cluster %s: %w", name, err)
	}

	delete(rcm.clusters, name)
	return nil
}

// GetAllClusters 获取所有集群
func (rcm *RedisClusterManager) GetAllClusters() map[string]*RedisCluster {
	rcm.mu.RLock()
	defer rcm.mu.RUnlock()

	clusters := make(map[string]*RedisCluster)
	for name, cluster := range rcm.clusters {
		clusters[name] = cluster
	}

	return clusters
}

// GetClusterStats 获取所有集群统计
func (rcm *RedisClusterManager) GetClusterStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	clusterStats := make(map[string]interface{})

	for name, cluster := range rcm.GetAllClusters() {
		clusterInfo, err := cluster.GetStats(ctx)
		if err != nil {
			log.Printf("Failed to get stats for cluster %s: %v", name, err)
			continue
		}
		clusterStats[name] = clusterInfo
	}

	stats["clusters"] = clusterStats
	stats["total_clusters"] = len(rcm.clusters)

	return stats, nil
}

// Close 关闭所有集群
func (rcm *RedisClusterManager) Close() error {
	rcm.mu.Lock()
	defer rcm.mu.Unlock()

	var lastErr error
	for name, cluster := range rcm.clusters {
		if err := cluster.Close(); err != nil {
			log.Printf("Failed to close cluster %s: %v", name, err)
			lastErr = err
		}
	}

	return lastErr
}

// RedisClusterFailover 集群故障转移
type RedisClusterFailover struct {
	manager *RedisClusterManager
	config  *FailoverConfig
}

// FailoverConfig 故障转移配置
type FailoverConfig struct {
	EnableFailover      bool          `json:"enable_failover"`
	FailoverTimeout     time.Duration `json:"failover_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxFailures         int           `json:"max_failures"`
}

// NewRedisClusterFailover 创建集群故障转移
func NewRedisClusterFailover(manager *RedisClusterManager, config *FailoverConfig) *RedisClusterFailover {
	return &RedisClusterFailover{
		manager: manager,
		config:  config,
	}
}

// CheckFailover 检查故障转移
func (rcf *RedisClusterFailover) CheckFailover(ctx context.Context) error {
	if !rcf.config.EnableFailover {
		return nil
	}

	clusters := rcf.manager.GetAllClusters()
	for name, cluster := range clusters {
		// 检查集群健康状态
		if err := rcf.checkClusterHealth(ctx, name, cluster); err != nil {
			log.Printf("Cluster %s health check failed: %v", name, err)
			// 这里可以实现故障转移逻辑
		}
	}

	return nil
}

// checkClusterHealth 检查集群健康状态
func (rcf *RedisClusterFailover) checkClusterHealth(ctx context.Context, name string, cluster *RedisCluster) error {
	// 简化的健康检查
	// 实际项目中应该实现更复杂的健康检查逻辑

	_, err := cluster.Get(ctx, "health_check")
	if err != nil {
		return fmt.Errorf("cluster %s is unhealthy: %w", name, err)
	}

	return nil
}

// RedisClusterMetrics 集群指标
type RedisClusterMetrics struct {
	TotalClusters    int                    `json:"total_clusters"`
	HealthyClusters  int                    `json:"healthy_clusters"`
	TotalNodes       int                    `json:"total_nodes"`
	TotalConnections int                    `json:"total_connections"`
	HitRate          float64                `json:"hit_rate"`
	MissRate         float64                `json:"miss_rate"`
	AvgResponseTime  time.Duration          `json:"avg_response_time"`
	ClusterStats     map[string]interface{} `json:"cluster_stats"`
}

// GetMetrics 获取集群指标
func (rcm *RedisClusterManager) GetMetrics(ctx context.Context) (*RedisClusterMetrics, error) {
	stats, err := rcm.GetClusterStats(ctx)
	if err != nil {
		return nil, err
	}

	metrics := &RedisClusterMetrics{
		TotalClusters:   len(rcm.clusters),
		HealthyClusters: len(rcm.clusters), // 简化实现
		ClusterStats:    stats,
	}

	// 计算节点总数
	if clusterMap, ok := stats["clusters"].(map[string]interface{}); ok {
		totalNodes := 0
		for _, clusterInfo := range clusterMap {
			if cluster, ok := clusterInfo.(map[string]interface{}); ok {
				if nodeCount, ok := cluster["node_count"].(int); ok {
					totalNodes += nodeCount
				}
			}
		}
		metrics.TotalNodes = totalNodes
	}

	return metrics, nil
}

// RedisClusterBalancer 集群负载均衡器
type RedisClusterBalancer struct {
	manager  *RedisClusterManager
	strategy LoadBalanceStrategy
}

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	LeastConnections
	WeightedRoundRobin
	Random
)

// NewRedisClusterBalancer 创建集群负载均衡器
func NewRedisClusterBalancer(manager *RedisClusterManager, strategy LoadBalanceStrategy) *RedisClusterBalancer {
	return &RedisClusterBalancer{
		manager:  manager,
		strategy: strategy,
	}
}

// GetCluster 根据负载均衡策略获取集群
func (rcb *RedisClusterBalancer) GetCluster(key string) (*RedisCluster, error) {
	clusters := rcb.manager.GetAllClusters()
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters available")
	}

	switch rcb.strategy {
	case RoundRobin:
		return rcb.roundRobin(clusters)
	case LeastConnections:
		return rcb.leastConnections(clusters)
	case WeightedRoundRobin:
		return rcb.weightedRoundRobin(clusters)
	case Random:
		return rcb.random(clusters)
	default:
		return rcb.roundRobin(clusters)
	}
}

// roundRobin 轮询策略
func (rcb *RedisClusterBalancer) roundRobin(clusters map[string]*RedisCluster) (*RedisCluster, error) {
	// 简化的轮询实现
	// 实际项目中应该维护轮询状态

	for _, cluster := range clusters {
		return cluster, nil
	}

	return nil, fmt.Errorf("no healthy cluster available")
}

// leastConnections 最少连接策略
func (rcb *RedisClusterBalancer) leastConnections(clusters map[string]*RedisCluster) (*RedisCluster, error) {
	// 简化的最少连接实现
	// 实际项目应该获取每个集群的连接数

	for _, cluster := range clusters {
		return cluster, nil
	}

	return nil, fmt.Errorf("no healthy cluster available")
}

// weightedRoundRobin 加权轮询策略
func (rcb *RedisClusterBalancer) weightedRoundRobin(clusters map[string]*RedisCluster) (*RedisCluster, error) {
	// 简化的加权轮询实现
	// 实际项目应该根据集群性能设置权重

	for _, cluster := range clusters {
		return cluster, nil
	}

	return nil, fmt.Errorf("no healthy cluster available")
}

// random 随机策略
func (rcb *RedisClusterBalancer) random(clusters map[string]*RedisCluster) (*RedisCluster, error) {
	// 简化的随机实现
	// 实际项目应该使用更好的随机算法

	for _, cluster := range clusters {
		return cluster, nil
	}

	return nil, fmt.Errorf("no healthy cluster available")
}
