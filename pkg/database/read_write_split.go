package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"

	"gorm.io/gorm"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	SSLMode  string `json:"ssl_mode"`
}

// ReadWriteSplit 读写分离
type ReadWriteSplit struct {
	masterDB     *gorm.DB
	slaveDBs     []*gorm.DB
	currentSlave int
	mu           sync.RWMutex
	healthCheck  *HealthChecker
	metrics      *metrics.MetricsCollector
	config       *ReadWriteSplitConfig
}

// ReadWriteSplitConfig 读写分离配置
type ReadWriteSplitConfig struct {
	SlaveCount          int           `json:"slave_count"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxRetries          int           `json:"max_retries"`
	FailoverTimeout     time.Duration `json:"failover_timeout"`
	EnableLoadBalance   bool          `json:"enable_load_balance"`
	ReadWeight          int           `json:"read_weight"`  // 读操作权重
	WriteWeight         int           `json:"write_weight"` // 写操作权重
}

// HealthChecker 健康检查器
type HealthChecker struct {
	slaves []*gorm.DB
	config *ReadWriteSplitConfig
	stopCh chan struct{}
	mu     sync.RWMutex
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(slaves []*gorm.DB, config *ReadWriteSplitConfig) *HealthChecker {
	return &HealthChecker{
		slaves: slaves,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start 开始健康检查
func (hc *HealthChecker) Start() {
	ticker := time.NewTicker(hc.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkAllSlaves()
		case <-hc.stopCh:
			return
		}
	}
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

// checkAllSlaves 检查所有从库健康状态
func (hc *HealthChecker) checkAllSlaves() {
	for i, slave := range hc.slaves {
		if slave == nil {
			continue
		}

		sqlDB, err := slave.DB()
		if err != nil {
			log.Printf("Failed to get slave DB %d: %v", i, err)
			continue
		}

		// 执行健康检查查询
		err = sqlDB.Ping()
		if err != nil {
			log.Printf("Slave %d health check failed: %v", i, err)
			// 这里可以实现故障转移逻辑
		}
	}
}

// GetHealthySlave 获取健康的从库
func (hc *HealthChecker) GetHealthySlave() *gorm.DB {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// 简单的轮询策略，实际项目中可以使用更复杂的负载均衡算法
	for _, slave := range hc.slaves {
		if slave != nil {
			sqlDB, err := slave.DB()
			if err == nil {
				if err := sqlDB.Ping(); err == nil {
					return slave
				}
			}
		}
	}

	return nil
}

// NewReadWriteSplit 创建读写分离实例
func NewReadWriteSplit(masterConfig DatabaseConfig, slaveConfigs []DatabaseConfig, metricsCollector *metrics.MetricsCollector) (*ReadWriteSplit, error) {
	// 连接主库
	masterDB := InitDatabase()
	if masterDB == nil {
		return nil, fmt.Errorf("failed to connect to master database")
	}

	// 连接从库
	var slaveDBs []*gorm.DB
	for range slaveConfigs {
		slaveDB := InitDatabase()
		if slaveDB == nil {
			log.Printf("Failed to connect to slave database")
			continue
		}
		slaveDBs = append(slaveDBs, slaveDB)
	}

	if len(slaveDBs) == 0 {
		return nil, fmt.Errorf("no available slave databases")
	}

	config := &ReadWriteSplitConfig{
		SlaveCount:          len(slaveDBs),
		HealthCheckInterval: time.Second * 30,
		MaxRetries:          3,
		FailoverTimeout:     time.Second * 5,
		EnableLoadBalance:   true,
		ReadWeight:          70, // 70% 读操作
		WriteWeight:         30, // 30% 写操作
	}

	rws := &ReadWriteSplit{
		masterDB:     masterDB,
		slaveDBs:     slaveDBs,
		currentSlave: 0,
		healthCheck:  NewHealthChecker(slaveDBs, config),
		metrics:      metricsCollector,
		config:       config,
	}

	// 启动健康检查
	go rws.healthCheck.Start()

	return rws, nil
}

// Master 获取主库连接（用于写操作）
func (rws *ReadWriteSplit) Master() *gorm.DB {
	return rws.masterDB
}

// Slave 获取从库连接（用于读操作）
func (rws *ReadWriteSplit) Slave() *gorm.DB {
	if !rws.config.EnableLoadBalance {
		return rws.slaveDBs[0]
	}

	rws.mu.Lock()
	defer rws.mu.Unlock()

	// 轮询选择从库
	slave := rws.healthCheck.GetHealthySlave()
	if slave == nil {
		// 如果没有健康的从库，回退到主库
		log.Println("No healthy slave available, falling back to master")
		return rws.masterDB
	}

	return slave
}

// GetDB 根据操作类型获取数据库连接
func (rws *ReadWriteSplit) GetDB(readOnly bool) *gorm.DB {
	if readOnly {
		return rws.Slave()
	}
	return rws.Master()
}

// ExecuteRead 执行读操作
func (rws *ReadWriteSplit) ExecuteRead(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db := rws.Slave()
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 记录读操作指标
	rws.metrics.RecordDBError("read_operation", "execute")
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rws.metrics.RecordDBQuery("read", "unknown", duration, true)
	}()

	return sqlDB.QueryContext(ctx, query, args...)
}

// ExecuteWrite 执行写操作
func (rws *ReadWriteSplit) ExecuteWrite(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	db := rws.Master()
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 记录写操作指标
	rws.metrics.RecordDBError("write_operation", "execute")
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rws.metrics.RecordDBQuery("write", "unknown", duration, true)
	}()

	return sqlDB.ExecContext(ctx, query, args...)
}

// Transaction 执行事务（在主库上）
func (rws *ReadWriteSplit) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return rws.masterDB.WithContext(ctx).Transaction(fn)
}

// BatchInsert 批量插入（在主库上）
func (rws *ReadWriteSplit) BatchInsert(ctx context.Context, tableName string, records []interface{}) error {
	return rws.masterDB.WithContext(ctx).Table(tableName).CreateInBatches(records, 100).Error
}

// BatchUpdate 批量更新（在主库上）
func (rws *ReadWriteSplit) BatchUpdate(ctx context.Context, tableName string, updates map[string]interface{}, conditions map[string]interface{}) error {
	return rws.masterDB.WithContext(ctx).Table(tableName).Where(conditions).Updates(updates).Error
}

// BatchDelete 批量删除（在主库上）
func (rws *ReadWriteSplit) BatchDelete(ctx context.Context, tableName string, conditions map[string]interface{}) error {
	return rws.masterDB.WithContext(ctx).Table(tableName).Where(conditions).Delete(nil).Error
}

// GetStats 获取读写分离统计信息
func (rws *ReadWriteSplit) GetStats() map[string]interface{} {
	rws.mu.RLock()
	defer rws.mu.RUnlock()

	stats := make(map[string]interface{})

	// 主库统计
	masterStats := rws.getDBStats(rws.masterDB)
	stats["master"] = masterStats

	// 从库统计
	slaveStats := make([]map[string]interface{}, len(rws.slaveDBs))
	for i, slave := range rws.slaveDBs {
		if slave != nil {
			slaveStats[i] = rws.getDBStats(slave)
		}
	}
	stats["slaves"] = slaveStats

	stats["current_slave"] = rws.currentSlave
	stats["total_slaves"] = len(rws.slaveDBs)

	return stats
}

// getDBStats 获取数据库统计信息
func (rws *ReadWriteSplit) getDBStats(db *gorm.DB) map[string]interface{} {
	sqlDB, err := db.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"open_connections":    stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration,
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}
}

// Close 关闭所有数据库连接
func (rws *ReadWriteSplit) Close() error {
	// 停止健康检查
	rws.healthCheck.Stop()

	// 关闭主库
	sqlDB, err := rws.masterDB.DB()
	if err != nil {
		log.Printf("Failed to get master database connection: %v", err)
	} else {
		if err := sqlDB.Close(); err != nil {
			log.Printf("Failed to close master database: %v", err)
		}
	}

	// 关闭从库
	for _, slave := range rws.slaveDBs {
		sqlDB, err := slave.DB()
		if err != nil {
			log.Printf("Failed to get slave database connection: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Failed to close slave database: %v", err)
			}
		}
	}

	return nil
}

// Failover 故障转移
func (rws *ReadWriteSplit) Failover() error {
	log.Println("Starting failover process...")

	// 尝试重新连接所有从库
	for i, slave := range rws.slaveDBs {
		if slave == nil {
			continue
		}

		// 这里应该有重新连接的逻辑
		// 简化实现，只是记录日志
		log.Printf("Checking slave %d for failover", i)
	}

	// 更新指标
	rws.metrics.RecordDBError("failover", "executed")

	return nil
}

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	LeastConnections
	WeightedRoundRobin
	Random
)

// SetLoadBalanceStrategy 设置负载均衡策略
func (rws *ReadWriteSplit) SetLoadBalanceStrategy(strategy LoadBalanceStrategy) {
	// 这里可以实现不同的负载均衡策略
	// 当前使用简单的轮询
}

// ReadOnlyConnection 只读连接包装器
type ReadOnlyConnection struct {
	db *gorm.DB
}

// NewReadOnlyConnection 创建只读连接
func NewReadOnlyConnection(db *gorm.DB) *ReadOnlyConnection {
	return &ReadOnlyConnection{db: db}
}

// Find 查询操作
func (roc *ReadOnlyConnection) Find(dest interface{}, conds ...interface{}) error {
	result := roc.db.Find(dest, conds...)
	return result.Error
}

// First 查询单条记录
func (roc *ReadOnlyConnection) First(dest interface{}, conds ...interface{}) error {
	result := roc.db.First(dest, conds...)
	return result.Error
}

// Count 计数操作
func (roc *ReadOnlyConnection) Count(count *int64, conds ...interface{}) error {
	result := roc.db.Model(&struct{}{}).Count(count)
	return result.Error
}

// WriteOnlyConnection 只写连接包装器
type WriteOnlyConnection struct {
	db *gorm.DB
}

// NewWriteOnlyConnection 创建只写连接
func NewWriteOnlyConnection(db *gorm.DB) *WriteOnlyConnection {
	return &WriteOnlyConnection{db: db}
}

// Create 创建记录
func (woc *WriteOnlyConnection) Create(value interface{}) error {
	return woc.db.Create(value).Error
}

// Update 更新记录
func (woc *WriteOnlyConnection) Update(value interface{}, conds ...interface{}) error {
	return woc.db.Model(value).Where(conds[0], conds[1:]...).Updates(value).Error
}

// Delete 删除记录
func (woc *WriteOnlyConnection) Delete(value interface{}, conds ...interface{}) error {
	return woc.db.Delete(value, conds...).Error
}

// QueryRouter 查询路由器
type QueryRouter struct {
	rws *ReadWriteSplit
}

// NewQueryRouter 创建查询路由器
func NewQueryRouter(rws *ReadWriteSplit) *QueryRouter {
	return &QueryRouter{rws: rws}
}

// RouteQuery 路由查询到合适的数据库
func (qr *QueryRouter) RouteQuery(query string) *gorm.DB {
	// 简化的查询路由逻辑
	// 实际项目中应该使用 SQL 解析器来准确判断查询类型

	query = strings.ToLower(query)

	// 写操作关键词
	writeKeywords := []string{"insert", "update", "delete", "create", "drop", "alter", "truncate"}

	for _, keyword := range writeKeywords {
		if strings.Contains(query, keyword) {
			return qr.rws.Master()
		}
	}

	// 默认使用从库进行读操作
	return qr.rws.Slave()
}

// Execute 执行查询
func (qr *QueryRouter) Execute(ctx context.Context, query string, args ...interface{}) error {
	db := qr.RouteQuery(query)
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	_, err = sqlDB.ExecContext(ctx, query, args...)
	return err
}

// Query 执行查询并返回结果
func (qr *QueryRouter) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db := qr.RouteQuery(query)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	return sqlDB.QueryContext(ctx, query, args...)
}

// ConnectionPool 连接池管理器
type ConnectionPool struct {
	masterPool *sql.DB
	slavePools []*sql.DB
	config     *ReadWriteSplitConfig
}

// NewConnectionPool 创建连接池管理器
func NewConnectionPool(rws *ReadWriteSplit) *ConnectionPool {
	masterSQLDB, _ := rws.masterDB.DB()

	var slaveSQLDBs []*sql.DB
	for _, slave := range rws.slaveDBs {
		if slave != nil {
			sqlDB, _ := slave.DB()
			slaveSQLDBs = append(slaveSQLDBs, sqlDB)
		}
	}

	return &ConnectionPool{
		masterPool: masterSQLDB,
		slavePools: slaveSQLDBs,
		config:     rws.config,
	}
}

// GetMasterPool 获取主库连接池
func (cp *ConnectionPool) GetMasterPool() *sql.DB {
	return cp.masterPool
}

// GetSlavePool 获取从库连接池
func (cp *ConnectionPool) GetSlavePool() *sql.DB {
	if len(cp.slavePools) == 0 {
		return cp.masterPool
	}

	// 简单的轮询策略
	index := int(time.Now().Nanosecond()) % len(cp.slavePools)
	return cp.slavePools[index]
}

// GetPoolStats 获取连接池统计
func (cp *ConnectionPool) GetPoolStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 主库统计
	stats["master"] = cp.masterPool.Stats()

	// 从库统计
	slaveStats := make([]interface{}, len(cp.slavePools))
	for i, pool := range cp.slavePools {
		if pool != nil {
			slaveStats[i] = pool.Stats()
		}
	}
	stats["slaves"] = slaveStats

	return stats
}

// Close 关闭所有连接池
func (cp *ConnectionPool) Close() error {
	var lastErr error

	// 关闭主库连接池
	if err := cp.masterPool.Close(); err != nil {
		lastErr = err
	}

	// 关闭从库连接池
	for i, pool := range cp.slavePools {
		if pool != nil {
			if err := pool.Close(); err != nil {
				lastErr = err
				log.Printf("Failed to close slave pool %d: %v", i, err)
			}
		}
	}

	return lastErr
}
