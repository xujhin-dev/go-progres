package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"user_crud_jwt/pkg/metrics"
)

// IndexOptimizer 索引优化器
type IndexOptimizer struct {
	db               *sql.DB
	metricsCollector *metrics.MetricsCollector
	queryAnalyzer    *QueryAnalyzer
	indexAnalyzer    *IndexAnalyzer
}

// QueryAnalyzer 查询分析器
type QueryAnalyzer struct {
	db *sql.DB
}

// IndexAnalyzer 索引分析器
type IndexAnalyzer struct {
	db *sql.DB
}

// NewIndexOptimizer 创建索引优化器
func NewIndexOptimizer(db *sql.DB, metricsCollector *metrics.MetricsCollector) *IndexOptimizer {
	return &IndexOptimizer{
		db:               db,
		metricsCollector: metricsCollector,
		queryAnalyzer:    NewQueryAnalyzer(db),
		indexAnalyzer:    NewIndexAnalyzer(db),
	}
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name        string    `json:"name"`
	Table       string    `json:"table"`
	Columns     []string  `json:"columns"`
	IsUnique    bool      `json:"is_unique"`
	IsPrimary   bool      `json:"is_primary"`
	Cardinality int64     `json:"cardinality"`
	Size        int64     `json:"size"`
	Usage       int64     `json:"usage"`
	LastUsed    time.Time `json:"last_used"`
}

// QueryPattern 查询模式
type QueryPattern struct {
	Table       string    `json:"table"`
	Columns     []string  `json:"columns"`
	WhereClause string    `json:"where_clause"`
	OrderBy     []string  `json:"order_by"`
	GroupBy     []string  `json:"group_by"`
	Frequency   int       `json:"frequency"`
	AvgTime     float64   `json:"avg_time"`
	LastSeen    time.Time `json:"last_seen"`
}

// IndexRecommendation 索引推荐
type IndexRecommendation struct {
	Table         string   `json:"table"`
	Columns       []string `json:"columns"`
	Type          string   `json:"type"` // btree, hash, gin, gist
	Reason        string   `json:"reason"`
	Impact        string   `json:"impact"`         // high, medium, low
	EstimatedGain float64  `json:"estimated_gain"` // 预估性能提升百分比
	Priority      int      `json:"priority"`
}

// NewQueryAnalyzer 创建查询分析器
func NewQueryAnalyzer(db *sql.DB) *QueryAnalyzer {
	return &QueryAnalyzer{db: db}
}

// AnalyzeQueries 分析查询模式
func (qa *QueryAnalyzer) AnalyzeQueries(ctx context.Context, duration time.Duration) ([]QueryPattern, error) {
	// 这里实现查询模式分析
	// 实际项目中可能需要解析慢查询日志或使用 pg_stat_statements

	query := `
		SELECT 
			query,
			calls,
			total_time,
			mean_time,
			rows
		FROM pg_stat_statements 
		WHERE calls > 10 
		ORDER BY total_time DESC 
		LIMIT 100
	`

	rows, err := qa.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze queries: %w", err)
	}
	defer rows.Close()

	var patterns []QueryPattern
	for rows.Next() {
		var queryText sql.NullString
		var calls int64
		var totalTime, meanTime float64
		var rowsReturned int64

		if err := rows.Scan(&queryText, &calls, &totalTime, &meanTime, &rowsReturned); err != nil {
			continue
		}

		if !queryText.Valid {
			continue
		}

		// 解析查询语句，提取表名和列名
		pattern := qa.parseQuery(queryText.String, int(calls), meanTime)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}

	return patterns, nil
}

// parseQuery 解析查询语句
func (qa *QueryAnalyzer) parseQuery(query string, frequency int, avgTime float64) *QueryPattern {
	// 简化的查询解析
	// 实际项目中应该使用 SQL 解析器

	var table string
	var columns []string
	var whereClause string

	// 提取表名
	if strings.Contains(query, "FROM") {
		parts := strings.Split(query, "FROM")
		if len(parts) > 1 {
			tablePart := strings.TrimSpace(parts[1])
			table = strings.Split(tablePart, " ")[0]
		}
	}

	// 提取 WHERE 条件
	if strings.Contains(query, "WHERE") {
		parts := strings.Split(query, "WHERE")
		if len(parts) > 1 {
			wherePart := strings.TrimSpace(parts[1])
			// 简化处理，取到下一个关键字之前
			if idx := strings.Index(wherePart, "ORDER BY"); idx != -1 {
				whereClause = strings.TrimSpace(wherePart[:idx])
			} else if idx := strings.Index(wherePart, "GROUP BY"); idx != -1 {
				whereClause = strings.TrimSpace(wherePart[:idx])
			} else if idx := strings.Index(wherePart, "LIMIT"); idx != -1 {
				whereClause = strings.TrimSpace(wherePart[:idx])
			} else {
				whereClause = wherePart
			}
		}
	}

	return &QueryPattern{
		Table:       table,
		Columns:     columns,
		WhereClause: whereClause,
		Frequency:   frequency,
		AvgTime:     avgTime,
		LastSeen:    time.Now(),
	}
}

// NewIndexAnalyzer 创建索引分析器
func NewIndexAnalyzer(db *sql.DB) *IndexAnalyzer {
	return &IndexAnalyzer{db: db}
}

// AnalyzeIndexes 分析现有索引
func (ia *IndexAnalyzer) AnalyzeIndexes(ctx context.Context) ([]IndexInfo, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			indexname,
			indexdef,
			indisunique,
			indisprimary
		FROM pg_indexes 
		WHERE schemaname = 'public'
		ORDER BY tablename, indexname
	`

	rows, err := ia.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze indexes: %w", err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var schema, table, name, definition string
		var isUnique, isPrimary bool

		if err := rows.Scan(&schema, &table, &name, &definition, &isUnique, &isPrimary); err != nil {
			continue
		}

		// 解析索引列
		columns := ia.parseIndexColumns(definition)

		// 获取索引统计信息
		cardinality, size, usage, lastUsed := ia.getIndexStats(ctx, schema, table, name)

		indexes = append(indexes, IndexInfo{
			Name:        name,
			Table:       table,
			Columns:     columns,
			IsUnique:    isUnique,
			IsPrimary:   isPrimary,
			Cardinality: cardinality,
			Size:        size,
			Usage:       usage,
			LastUsed:    lastUsed,
		})
	}

	return indexes, nil
}

// parseIndexColumns 解析索引列
func (ia *IndexAnalyzer) parseIndexColumns(definition string) []string {
	// 简化的列名解析
	var columns []string

	// 查找括号内的内容
	start := strings.Index(definition, "(")
	end := strings.LastIndex(definition, ")")

	if start != -1 && end != -1 && end > start {
		columnsStr := definition[start+1 : end]
		columns = strings.Split(columnsStr, ",")

		// 清理列名
		for i, col := range columns {
			columns[i] = strings.TrimSpace(strings.ReplaceAll(col, "\"", ""))
		}
	}

	return columns
}

// getIndexStats 获取索引统计信息
func (ia *IndexAnalyzer) getIndexStats(ctx context.Context, schema, table, index string) (int64, int64, int64, time.Time) {
	// 获取索引使用统计
	usageQuery := `
		SELECT idx_scan, idx_tup_read, idx_tup_fetch
		FROM pg_stat_user_indexes 
		WHERE schemaname = $1 AND tablename = $2 AND indexrelname = $3
	`

	var scans, reads, fetches int64
	err := ia.db.QueryRowContext(ctx, usageQuery, schema, table, index).Scan(&scans, &reads, &fetches)
	if err != nil {
		return 0, 0, 0, time.Time{}
	}

	// 获取索引大小
	sizeQuery := `
		SELECT pg_relation_size(indexrelid)
		FROM pg_stat_user_indexes 
		WHERE schemaname = $1 AND tablename = $2 AND indexrelname = $3
	`

	var size int64
	err = ia.db.QueryRowContext(ctx, sizeQuery, schema, table, index).Scan(&size)
	if err != nil {
		return 0, 0, 0, time.Time{}
	}

	// 获取基数（近似值）
	cardinalityQuery := `
		SELECT n_distinct
		FROM pg_stats 
		WHERE schemaname = $1 AND tablename = $2 AND attname = $3
		LIMIT 1
	`

	var cardinality int64
	if len(ia.parseIndexColumns(fmt.Sprintf("CREATE INDEX %s ON %s", index, table))) > 0 {
		firstColumn := ia.parseIndexColumns(fmt.Sprintf("CREATE INDEX %s ON %s", index, table))[0]
		ia.db.QueryRowContext(ctx, cardinalityQuery, schema, table, firstColumn).Scan(&cardinality)
	}

	return cardinality, size, scans, time.Now()
}

// OptimizeIndexes 优化索引
func (io *IndexOptimizer) OptimizeIndexes(ctx context.Context) ([]IndexRecommendation, error) {
	// 分析查询模式
	queryPatterns, err := io.queryAnalyzer.AnalyzeQueries(ctx, time.Hour*24)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze queries: %w", err)
	}

	// 分析现有索引
	existingIndexes, err := io.indexAnalyzer.AnalyzeIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze indexes: %w", err)
	}

	// 生成索引推荐
	recommendations := io.generateRecommendations(queryPatterns, existingIndexes)

	return recommendations, nil
}

// generateRecommendations 生成索引推荐
func (io *IndexOptimizer) generateRecommendations(queryPatterns []QueryPattern, existingIndexes []IndexInfo) []IndexRecommendation {
	var recommendations []IndexRecommendation

	// 按表分组查询模式
	queriesByTable := make(map[string][]QueryPattern)
	for _, pattern := range queryPatterns {
		queriesByTable[pattern.Table] = append(queriesByTable[pattern.Table], pattern)
	}

	// 为每个表生成推荐
	for table, patterns := range queriesByTable {
		tableRecommendations := io.analyzeTableQueries(table, patterns, existingIndexes)
		recommendations = append(recommendations, tableRecommendations...)
	}

	// 分析未使用的索引
	unusedRecommendations := io.analyzeUnusedIndexes(existingIndexes)
	recommendations = append(recommendations, unusedRecommendations...)

	// 按优先级排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})

	return recommendations
}

// analyzeTableQueries 分析表查询
func (io *IndexOptimizer) analyzeTableQueries(table string, patterns []QueryPattern, existingIndexes []IndexInfo) []IndexRecommendation {
	var recommendations []IndexRecommendation

	// 创建现有索引映射
	indexMap := make(map[string][]string)
	for _, index := range existingIndexes {
		if index.Table == table {
			indexMap[index.Name] = index.Columns
		}
	}

	// 分析高频查询
	for _, pattern := range patterns {
		if pattern.Frequency < 10 {
			continue // 跳过低频查询
		}

		// 提取 WHERE 条件中的列
		whereColumns := io.extractWhereColumns(pattern.WhereClause)

		// 检查是否已有合适的索引
		hasIndex := io.checkExistingIndex(whereColumns, indexMap)

		if !hasIndex && len(whereColumns) > 0 {
			recommendation := IndexRecommendation{
				Table:         table,
				Columns:       whereColumns,
				Type:          "btree",
				Reason:        fmt.Sprintf("高频查询（%d次/天）缺少索引", pattern.Frequency),
				Impact:        io.calculateImpact(pattern),
				EstimatedGain: io.estimateGain(pattern),
				Priority:      io.calculatePriority(pattern),
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

// extractWhereColumns 提取 WHERE 条件中的列
func (io *IndexOptimizer) extractWhereColumns(whereClause string) []string {
	var columns []string

	// 简化的 WHERE 条件解析
	// 实际项目中应该使用 SQL 解析器

	if whereClause == "" {
		return columns
	}

	// 移除括号和逻辑运算符
	cleanWhere := strings.ReplaceAll(whereClause, "(", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, ")", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, "AND", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, "OR", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, "=", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, ">", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, "<", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, "LIKE", " ")
	cleanWhere = strings.ReplaceAll(cleanWhere, "IN", " ")

	words := strings.Fields(cleanWhere)
	for _, word := range words {
		// 简单的列名判断（不包含数字和特殊字符）
		if io.isColumnName(word) {
			columns = append(columns, word)
		}
	}

	// 去重
	uniqueColumns := make(map[string]bool)
	var result []string
	for _, col := range columns {
		if !uniqueColumns[col] {
			uniqueColumns[col] = true
			result = append(result, col)
		}
	}

	return result
}

// isColumnName 判断是否为列名
func (io *IndexOptimizer) isColumnName(word string) bool {
	// 简单的列名判断
	if len(word) == 0 {
		return false
	}

	// 不包含数字（排除常量）
	for _, r := range word {
		if r >= '0' && r <= '9' {
			return false
		}
	}

	// 不包含常见的 SQL 关键字
	keywords := []string{"true", "false", "null", "undefined"}
	for _, keyword := range keywords {
		if strings.ToLower(word) == keyword {
			return false
		}
	}

	return true
}

// checkExistingIndex 检查是否已有合适的索引
func (io *IndexOptimizer) checkExistingIndex(columns []string, indexMap map[string][]string) bool {
	for _, indexColumns := range indexMap {
		if io.indexMatches(columns, indexColumns) {
			return true
		}
	}
	return false
}

// indexMatches 检查索引是否匹配查询需求
func (io *IndexOptimizer) indexMatches(queryColumns, indexColumns []string) bool {
	if len(indexColumns) == 0 {
		return false
	}

	// 检查索引是否包含查询的主要列
	for i, col := range queryColumns {
		if i >= len(indexColumns) {
			break
		}
		if indexColumns[i] != col {
			return false
		}
	}

	return true
}

// calculateImpact 计算影响级别
func (io *IndexOptimizer) calculateImpact(pattern QueryPattern) string {
	if pattern.AvgTime > 1000 { // 超过1秒
		return "high"
	} else if pattern.AvgTime > 500 { // 超过500ms
		return "medium"
	}
	return "low"
}

// estimateGain 估算性能提升
func (io *IndexOptimizer) estimateGain(pattern QueryPattern) float64 {
	// 简化的性能提升估算
	// 实际项目中应该基于基准测试

	baseTime := pattern.AvgTime
	if baseTime < 100 {
		return 10.0 // 少于100ms的查询提升有限
	} else if baseTime < 500 {
		return 30.0 // 100-500ms的查询可能有30%提升
	} else if baseTime < 1000 {
		return 50.0 // 500ms-1s的查询可能有50%提升
	} else {
		return 70.0 // 超过1s的查询可能有70%提升
	}
}

// calculatePriority 计算优先级
func (io *IndexOptimizer) calculatePriority(pattern QueryPattern) int {
	priority := 0

	// 基于频率
	if pattern.Frequency > 1000 {
		priority += 30
	} else if pattern.Frequency > 100 {
		priority += 20
	} else if pattern.Frequency > 10 {
		priority += 10
	}

	// 基于平均时间
	if pattern.AvgTime > 1000 {
		priority += 30
	} else if pattern.AvgTime > 500 {
		priority += 20
	} else if pattern.AvgTime > 100 {
		priority += 10
	}

	// 基于最近执行时间
	if time.Since(pattern.LastSeen) < time.Hour*24 {
		priority += 10
	}

	return priority
}

// analyzeUnusedIndexes 分析未使用的索引
func (io *IndexOptimizer) analyzeUnusedIndexes(existingIndexes []IndexInfo) []IndexRecommendation {
	var recommendations []IndexRecommendation

	for _, index := range existingIndexes {
		// 跳过主键索引
		if index.IsPrimary {
			continue
		}

		// 检查索引使用情况
		if index.Usage == 0 && time.Since(index.LastUsed) > time.Hour*24*7 {
			recommendation := IndexRecommendation{
				Table:         index.Table,
				Columns:       index.Columns,
				Type:          "drop",
				Reason:        fmt.Sprintf("索引 %s 在过去7天内未被使用", index.Name),
				Impact:        "medium",
				EstimatedGain: float64(index.Size) / 1024 / 1024, // 节省的MB空间
				Priority:      20,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

// CreateIndex 创建索引
func (io *IndexOptimizer) CreateIndex(ctx context.Context, recommendation IndexRecommendation) error {
	if recommendation.Type == "drop" {
		return io.dropIndex(ctx, recommendation)
	}

	var indexName string
	if len(recommendation.Columns) > 0 {
		indexName = fmt.Sprintf("idx_%s_%s", recommendation.Table, strings.Join(recommendation.Columns, "_"))
	} else {
		indexName = fmt.Sprintf("idx_%s_auto", recommendation.Table)
	}

	var createSQL string
	switch recommendation.Type {
	case "btree":
		createSQL = fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s)",
			indexName, recommendation.Table, strings.Join(recommendation.Columns, ", "))
	case "hash":
		createSQL = fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s USING hash (%s)",
			indexName, recommendation.Table, strings.Join(recommendation.Columns, ", "))
	default:
		createSQL = fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s (%s)",
			indexName, recommendation.Table, strings.Join(recommendation.Columns, ", "))
	}

	_, err := io.db.ExecContext(ctx, createSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 记录指标
	io.metricsCollector.RecordDBError("index_created", recommendation.Table)

	log.Printf("Created index: %s on table %s", indexName, recommendation.Table)
	return nil
}

// dropIndex 删除索引
func (io *IndexOptimizer) dropIndex(ctx context.Context, recommendation IndexRecommendation) error {
	indexName := fmt.Sprintf("idx_%s_%s", recommendation.Table, strings.Join(recommendation.Columns, "_"))

	dropSQL := fmt.Sprintf("DROP INDEX CONCURRENTLY %s", indexName)

	_, err := io.db.ExecContext(ctx, dropSQL)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	// 记录指标
	io.metricsCollector.RecordDBError("index_dropped", recommendation.Table)

	log.Printf("Dropped index: %s from table %s", indexName, recommendation.Table)
	return nil
}

// GetIndexStats 获取索引统计信息
func (io *IndexOptimizer) GetIndexStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_scan,
			idx_tup_read,
			idx_tup_fetch,
			pg_relation_size(indexrelid) as size
		FROM pg_stat_user_indexes
		WHERE schemaname = 'public'
		ORDER BY idx_scan DESC
	`

	rows, err := io.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	var totalScans, totalSize int64
	var indexCount int

	for rows.Next() {
		var schema, table, name string
		var scans, reads, fetches, size int64

		if err := rows.Scan(&schema, &table, &name, &scans, &reads, &fetches, &size); err != nil {
			continue
		}

		indexCount++
		totalScans += scans
		totalSize += size

		// 记录每个索引的统计
		indexKey := fmt.Sprintf("%s.%s", table, name)
		stats[indexKey] = map[string]interface{}{
			"scans":   scans,
			"reads":   reads,
			"fetches": fetches,
			"size":    size,
		}
	}

	stats["total_indexes"] = indexCount
	stats["total_scans"] = totalScans
	stats["total_size"] = totalSize
	stats["avg_scans_per_index"] = float64(totalScans) / float64(indexCount)

	return stats, nil
}

// RebuildIndex 重建索引
func (io *IndexOptimizer) RebuildIndex(ctx context.Context, tableName, indexName string) error {
	rebuildSQL := fmt.Sprintf("REINDEX INDEX %s ON %s", indexName, tableName)

	_, err := io.db.ExecContext(ctx, rebuildSQL)
	if err != nil {
		return fmt.Errorf("failed to rebuild index: %w", err)
	}

	// 记录指标
	io.metricsCollector.RecordDBError("index_rebuilt", tableName)

	log.Printf("Rebuilt index: %s on table %s", indexName, tableName)
	return nil
}

// AnalyzeTable 分析表统计信息
func (io *IndexOptimizer) AnalyzeTable(ctx context.Context, tableName string) error {
	analyzeSQL := fmt.Sprintf("ANALYZE %s", tableName)

	_, err := io.db.ExecContext(ctx, analyzeSQL)
	if err != nil {
		return fmt.Errorf("failed to analyze table: %w", err)
	}

	// 记录指标
	io.metricsCollector.RecordDBError("table_analyzed", tableName)

	log.Printf("Analyzed table: %s", tableName)
	return nil
}
