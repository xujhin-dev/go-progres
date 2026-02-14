package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"

	"gorm.io/gorm"
)

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	db               *gorm.DB
	metricsCollector *metrics.MetricsCollector
	slowQueryLog     *SlowQueryLog
	analyzer         *QueryOptimizerAnalyzer
	optimizer        *QueryOptimizerEngine
}

// SlowQueryLog 慢查询日志
type SlowQueryLog struct {
	queries []SlowQuery
	mu      sync.RWMutex
}

// SlowQuery 慢查询记录
type SlowQuery struct {
	ID          string        `json:"id"`
	Query       string        `json:"query"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
	Database    string        `json:"database"`
	Table       string        `json:"table"`
	Operation   string        `json:"operation"`
	Params      []interface{} `json:"params"`
	ExplainPlan []ExplainStep `json:"explain_plan"`
	Optimized   bool          `json:"optimized"`
}

// ExplainStep 执行计划步骤
type ExplainStep struct {
	Step        string  `json:"step"`
	Cost        float64 `json:"cost"`
	Rows        int64   `json:"rows"`
	Width       int64   `json:"width"`
	Description string  `json:"description"`
}

// QueryOptimizerAnalyzer 查询分析器
type QueryOptimizerAnalyzer struct {
	db *gorm.DB
}

// QueryOptimizerEngine 查询优化引擎
type QueryOptimizerEngine struct {
	db *gorm.DB
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *gorm.DB, metricsCollector *metrics.MetricsCollector) *QueryOptimizer {
	return &QueryOptimizer{
		db:               db,
		metricsCollector: metricsCollector,
		slowQueryLog:     NewSlowQueryLog(),
		analyzer:         NewQueryOptimizerAnalyzer(db),
		optimizer:        NewQueryOptimizerEngine(db),
	}
}

// NewSlowQueryLog 创建慢查询日志
func NewSlowQueryLog() *SlowQueryLog {
	return &SlowQueryLog{
		queries: make([]SlowQuery, 0),
	}
}

// NewQueryAnalyzer 创建查询分析器
func NewQueryOptimizerAnalyzer(db *gorm.DB) *QueryOptimizerAnalyzer {
	return &QueryOptimizerAnalyzer{db: db}
}

// NewQueryOptimizerEngine 创建查询优化引擎
func NewQueryOptimizerEngine(db *gorm.DB) *QueryOptimizerEngine {
	return &QueryOptimizerEngine{db: db}
}

// AnalyzeQuery 分析查询
func (qa *QueryOptimizerAnalyzer) AnalyzeQuery(ctx context.Context, query string, params []interface{}) (*QueryAnalysis, error) {
	analysis := &QueryAnalysis{
		Query:     query,
		Params:    params,
		Timestamp: time.Now(),
	}

	// 分析查询类型
	analysis.QueryType = qa.detectQueryType(query)

	// 提取表名
	analysis.Tables = qa.extractTables(query)

	// 提取字段
	analysis.Fields = qa.extractFields(query)

	// 分析 WHERE 条件
	analysis.WhereConditions = qa.extractWhereConditions(query)

	// 分析 JOIN 操作
	analysis.Joins = qa.extractJoins(query)

	// 分析聚合操作
	analysis.Aggregations = qa.extractAggregations(query)

	// 分析排序
	analysis.OrderBy = qa.extractOrderBy(query)

	// 分析分组
	analysis.GroupBy = qa.extractGroupBy(query)

	// 分析 LIMIT
	analysis.Limit = qa.extractLimit(query)

	// 计算复杂度
	analysis.Complexity = qa.calculateComplexity(analysis)

	return analysis, nil
}

// QueryAnalysis 查询分析结果
type QueryAnalysis struct {
	Query           string        `json:"query"`
	Params          []interface{} `json:"params"`
	QueryType       string        `json:"query_type"`
	Tables          []string      `json:"tables"`
	Fields          []string      `json:"fields"`
	WhereConditions []string      `json:"where_conditions"`
	Joins           []JoinInfo    `json:"joins"`
	Aggregations    []string      `json:"aggregations"`
	OrderBy         []string      `json:"order_by"`
	GroupBy         []string      `json:"group_by"`
	Limit           int           `json:"limit"`
	Complexity      int           `json:"complexity"`
	Timestamp       time.Time     `json:"timestamp"`
}

// JoinInfo JOIN 信息
type JoinInfo struct {
	Type       string `json:"type"`
	LeftTable  string `json:"left_table"`
	RightTable string `json:"right_table"`
	Condition  string `json:"condition"`
}

// detectQueryType 检测查询类型
func (qa *QueryAnalyzer) detectQueryType(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))

	if strings.HasPrefix(query, "select") {
		return "SELECT"
	} else if strings.HasPrefix(query, "insert") {
		return "INSERT"
	} else if strings.HasPrefix(query, "update") {
		return "UPDATE"
	} else if strings.HasPrefix(query, "delete") {
		return "DELETE"
	} else if strings.HasPrefix(query, "create") {
		return "CREATE"
	} else if strings.HasPrefix(query, "drop") {
		return "DROP"
	} else if strings.HasPrefix(query, "alter") {
		return "ALTER"
	}

	return "UNKNOWN"
}

// extractTables 提取表名
func (qa *QueryAnalyzer) extractTables(query string) []string {
	var tables []string

	// 简化的表名提取
	// 实际项目中应该使用 SQL 解析器

	// FROM 子句
	if strings.Contains(query, "from") {
		parts := strings.Split(query, "from")
		if len(parts) > 1 {
			fromPart := strings.TrimSpace(parts[1])
			tableName := strings.Split(fromPart, " ")[0]
			tables = append(tables, strings.Trim(tableName, "\"`"))
		}
	}

	// JOIN 子句
	joinKeywords := []string{"join", "inner join", "left join", "right join", "full join"}
	for _, keyword := range joinKeywords {
		if strings.Contains(query, keyword) {
			parts := strings.Split(query, keyword)
			if len(parts) > 1 {
				joinPart := strings.TrimSpace(parts[1])
				tableName := strings.Split(joinPart, " ")[0]
				tables = append(tables, strings.Trim(tableName, "\"`"))
			}
		}
	}

	// 去重
	uniqueTables := make(map[string]bool)
	var result []string
	for _, table := range tables {
		if table != "" && !uniqueTables[table] {
			uniqueTables[table] = true
			result = append(result, table)
		}
	}

	return result
}

// extractFields 提取字段
func (qa *QueryAnalyzer) extractFields(query string) []string {
	var fields []string

	// 简化的字段提取
	if strings.Contains(query, "select") {
		parts := strings.Split(query, "select")
		if len(parts) > 1 {
			selectPart := strings.TrimSpace(parts[1])

			// 找到 FROM 之前的部分
			fromIndex := strings.Index(selectPart, "from")
			if fromIndex != -1 {
				selectPart = selectPart[:fromIndex]
			}

			// 处理 SELECT *
			if strings.Contains(selectPart, "*") {
				fields = append(fields, "*")
			} else {
				// 分割字段
				fieldParts := strings.Split(selectPart, ",")
				for _, field := range fieldParts {
					field = strings.TrimSpace(field)
					// 移除函数调用
					if idx := strings.Index(field, "("); idx != -1 {
						field = field[:idx]
					}
					if field != "" {
						fields = append(fields, strings.Trim(field, "\"`"))
					}
				}
			}
		}
	}

	return fields
}

// extractWhereConditions 提取 WHERE 条件
func (qa *QueryAnalyzer) extractWhereConditions(query string) []string {
	var conditions []string

	if strings.Contains(query, "where") {
		parts := strings.Split(query, "where")
		if len(parts) > 1 {
			wherePart := strings.TrimSpace(parts[1])

			// 找到下一个关键字之前
			keywords := []string{"order by", "group by", "limit", "having", "offset"}
			for _, keyword := range keywords {
				if idx := strings.Index(wherePart, keyword); idx != -1 {
					wherePart = wherePart[:idx]
					break
				}
			}

			// 简化处理，直接返回整个 WHERE 子句
			if wherePart != "" {
				conditions = append(conditions, strings.TrimSpace(wherePart))
			}
		}
	}

	return conditions
}

// extractJoins 提取 JOIN 信息
func (qa *QueryAnalyzer) extractJoins(query string) []JoinInfo {
	var joins []JoinInfo

	joinKeywords := []string{"join", "inner join", "left join", "right join", "full join"}
	for _, keyword := range joinKeywords {
		if strings.Contains(query, keyword) {
			parts := strings.Split(query, keyword)
			if len(parts) > 1 {
				joinPart := strings.TrimSpace(parts[1])

				// 提取表名和条件
				tableName := strings.Split(joinPart, " ")[0]
				var condition string

				if strings.Contains(joinPart, "on") {
					onParts := strings.Split(joinPart, "on")
					if len(onParts) > 1 {
						condition = strings.TrimSpace(onParts[1])
					}
				}

				joins = append(joins, JoinInfo{
					Type:       strings.ToUpper(keyword),
					RightTable: strings.Trim(tableName, "\"`"),
					Condition:  condition,
				})
			}
		}
	}

	return joins
}

// extractAggregations 提取聚合操作
func (qa *QueryAnalyzer) extractAggregations(query string) []string {
	var aggregations []string

	aggFunctions := []string{"count", "sum", "avg", "min", "max", "stddev", "variance"}

	for _, funcName := range aggFunctions {
		if strings.Contains(strings.ToLower(query), funcName) {
			aggregations = append(aggregations, strings.ToUpper(funcName))
		}
	}

	return aggregations
}

// extractOrderBy 提取排序字段
func (qa *QueryAnalyzer) extractOrderBy(query string) []string {
	var orderFields []string

	if strings.Contains(query, "order by") {
		parts := strings.Split(query, "order by")
		if len(parts) > 1 {
			orderByPart := strings.TrimSpace(parts[1])

			// 找到下一个关键字之前
			keywords := []string{"limit", "offset", "having"}
			for _, keyword := range keywords {
				if idx := strings.Index(orderByPart, keyword); idx != -1 {
					orderByPart = orderByPart[:idx]
					break
				}
			}

			// 分割排序字段
			fieldParts := strings.Split(orderByPart, ",")
			for _, field := range fieldParts {
				field = strings.TrimSpace(field)
				// 移除排序方向
				if strings.HasSuffix(field, " desc") || strings.HasSuffix(field, " asc") {
					field = field[:len(field)-4]
				}
				if field != "" {
					orderFields = append(orderFields, strings.Trim(field, "\"`"))
				}
			}
		}
	}

	return orderFields
}

// extractGroupBy 提取分组字段
func (qa *QueryAnalyzer) extractGroupBy(query string) []string {
	var groupFields []string

	if strings.Contains(query, "group by") {
		parts := strings.Split(query, "group by")
		if len(parts) > 1 {
			groupByPart := strings.TrimSpace(parts[1])

			// 找到下一个关键字之前
			keywords := []string{"order by", "limit", "offset", "having"}
			for _, keyword := range keywords {
				if idx := strings.Index(groupByPart, keyword); idx != -1 {
					groupByPart = groupByPart[:idx]
					break
				}
			}

			// 分割分组字段
			fieldParts := strings.Split(groupByPart, ",")
			for _, field := range fieldParts {
				field = strings.TrimSpace(field)
				if field != "" {
					groupFields = append(groupFields, strings.Trim(field, "\"`"))
				}
			}
		}
	}

	return groupFields
}

// extractLimit 提取 LIMIT 值
func (qa *QueryAnalyzer) extractLimit(query string) int {
	if strings.Contains(query, "limit") {
		parts := strings.Split(query, "limit")
		if len(parts) > 1 {
			limitPart := strings.TrimSpace(parts[1])

			// 提取数字
			re := regexp.MustCompile(`\d+`)
			matches := re.FindString(limitPart)
			if matches != "" {
				if limit, err := fmt.Sscanf(matches, "%d", new(int)); err == nil {
					return limit
				}
			}
		}
	}

	return 0
}

// calculateComplexity 计算查询复杂度
func (qa *QueryAnalyzer) calculateComplexity(analysis *QueryAnalysis) int {
	complexity := 1

	// 基础复杂度
	complexity += len(analysis.Tables)
	complexity += len(analysis.Joins) * 2
	complexity += len(analysis.WhereConditions)
	complexity += len(analysis.Aggregations) * 2

	// 特殊操作增加复杂度
	if analysis.QueryType == "SELECT" {
		if len(analysis.Aggregations) > 0 {
			complexity += 5
		}
		if len(analysis.GroupBy) > 0 {
			complexity += 3
		}
		if len(analysis.OrderBy) > 0 {
			complexity += 2
		}
	}

	return complexity
}

// OptimizeQuery 优化查询
func (qoe *QueryOptimizerEngine) OptimizeQuery(analysis *QueryAnalysis) (*OptimizedQuery, error) {
	optimized := &OptimizedQuery{
		OriginalQuery:  analysis.Query,
		OptimizedQuery: analysis.Query,
		Optimizations:  []string{},
	}

	// 添加索引建议
	if len(analysis.WhereConditions) > 0 {
		indexSuggestion := qoe.suggestIndex(analysis)
		if indexSuggestion != "" {
			optimized.Optimizations = append(optimized.Optimizations, indexSuggestion)
		}
	}

	// 优化 JOIN 顺序
	if len(analysis.Joins) > 0 {
		optimized.OptimizedQuery = qoe.optimizeJoins(analysis)
		optimized.Optimizations = append(optimized.Optimizations, "Optimized JOIN order")
	}

	// 优化 WHERE 条件
	if len(analysis.WhereConditions) > 0 {
		optimized.OptimizedQuery = qoe.optimizeWhere(analysis)
		optimized.Optimizations = append(optimized.Optimizations, "Optimized WHERE conditions")
	}

	// 优化 LIMIT
	if analysis.Limit > 1000 {
		optimized.OptimizedQuery = qoe.optimizeLimit(analysis)
		optimized.Optimizations = append(optimized.Optimizations, "Optimized LIMIT clause")
	}

	return optimized, nil
}

// OptimizedQuery 优化后的查询
type OptimizedQuery struct {
	OriginalQuery  string   `json:"original_query"`
	OptimizedQuery string   `json:"optimized_query"`
	Optimizations  []string `json:"optimizations"`
	EstimatedGain  float64  `json:"estimated_gain"`
}

// suggestIndex 建议索引
func (qoe *QueryOptimizerEngine) suggestIndex(analysis *QueryAnalysis) string {
	if len(analysis.WhereConditions) > 0 {
		return fmt.Sprintf("Consider adding index on %s for WHERE conditions", strings.Join(analysis.WhereConditions, ", "))
	}
	return ""
}

// optimizeJoins 优化 JOIN
func (qoe *QueryOptimizerEngine) optimizeJoins(analysis *QueryAnalysis) string {
	// 简化的 JOIN 优化
	// 实际项目中应该根据表大小和连接条件优化 JOIN 顺序

	query := analysis.Query

	// 简单的优化：确保小表在前
	// 这里只是示例，实际实现会更复杂

	return query
}

// optimizeWhere 优化 WHERE 条件
func (qoe *QueryOptimizerEngine) optimizeWhere(analysis *QueryAnalysis) string {
	// 简化的 WHERE 优化
	// 实际项目中应该重写 WHERE 条件以提高性能

	query := analysis.Query

	// 简单的优化：将选择性高的条件放在前面
	// 这里只是示例，实际实现会更复杂

	return query
}

// optimizeLimit 优化 LIMIT
func (qoe *QueryOptimizerEngine) optimizeLimit(analysis *QueryAnalysis) string {
	query := analysis.Query

	// 简单的 LIMIT 优化：使用合理的 LIMIT 值
	if analysis.Limit > 1000 {
		query = strings.Replace(query, fmt.Sprintf("LIMIT %d", analysis.Limit), "LIMIT 1000", 1)
	}

	return query
}

// ExecuteQuery 执行查询并监控
func (qo *QueryOptimizer) ExecuteQuery(ctx context.Context, query string, params ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	// 记录查询开始
	qo.metricsCollector.RecordDBError("query_start", "execute")

	// 执行查询
	sqlDB, err := qo.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		qo.metricsCollector.RecordDBError("query_error", "execute")
		return nil, err
	}

	// 记录查询完成
	duration := time.Since(start)
	qo.metricsCollector.RecordDBQuery("query", "unknown", duration, true)

	// 检查是否为慢查询
	if duration > time.Millisecond*100 { // 超过100ms
		qo.slowQueryLog.AddSlowQuery(query, duration, params)
	}

	return rows, nil
}

// AddSlowQuery 添加慢查询记录
func (sql *SlowQueryLog) AddSlowQuery(query string, duration time.Duration, params []interface{}) {
	sql.mu.Lock()
	defer sql.mu.Unlock()

	slowQuery := SlowQuery{
		ID:        generateQueryID(),
		Query:     query,
		Duration:  duration,
		Timestamp: time.Now(),
		Params:    params,
		Optimized: false,
	}

	sql.queries = append(sql.queries, slowQuery)

	// 保持最近1000条记录
	if len(sql.queries) > 1000 {
		sql.queries = sql.queries[len(sql.queries)-1000:]
	}
}

// generateQueryID 生成查询ID
func generateQueryID() string {
	return fmt.Sprintf("query_%d", time.Now().UnixNano())
}

// GetSlowQueries 获取慢查询列表
func (sql *SlowQueryLog) GetSlowQueries(limit int) []SlowQuery {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	if limit <= 0 || limit > len(sql.queries) {
		limit = len(sql.queries)
	}

	// 按时间倒序排列
	sortedQueries := make([]SlowQuery, len(sql.queries))
	copy(sortedQueries, sql.queries)

	sort.Slice(sortedQueries, func(i, j int) bool {
		return sortedQueries[i].Timestamp.After(sortedQueries[j].Timestamp)
	})

	return sortedQueries[:limit]
}

// GetSlowQueryStats 获取慢查询统计
func (sql *SlowQueryLog) GetSlowQueryStats() map[string]interface{} {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	stats := make(map[string]interface{})

	if len(sql.queries) == 0 {
		return stats
	}

	var totalDuration time.Duration
	var maxDuration time.Duration
	var minDuration time.Duration = time.Hour * 24 // 初始化为24小时

	queryTypes := make(map[string]int)
	tables := make(map[string]int)

	for _, query := range sql.queries {
		totalDuration += query.Duration

		if query.Duration > maxDuration {
			maxDuration = query.Duration
		}
		if query.Duration < minDuration {
			minDuration = query.Duration
		}

		// 统计查询类型
		analysis, err := NewQueryOptimizerAnalyzer(nil).AnalyzeQuery(context.Background(), query.Query, query.Params)
		if err == nil {
			queryTypes[analysis.QueryType]++
			for _, table := range analysis.Tables {
				tables[table]++
			}
		}
	}

	stats["total_queries"] = len(sql.queries)
	stats["total_duration"] = totalDuration.String()
	stats["max_duration"] = maxDuration.String()
	stats["min_duration"] = minDuration.String()
	stats["avg_duration"] = (totalDuration / time.Duration(len(sql.queries))).String()
	stats["query_types"] = queryTypes
	stats["tables"] = tables

	return stats
}

// AnalyzeSlowQueries 分析慢查询
func (qo *QueryOptimizer) AnalyzeSlowQueries(ctx context.Context) ([]OptimizationSuggestion, error) {
	slowQueries := qo.slowQueryLog.GetSlowQueries(50)
	var suggestions []OptimizationSuggestion

	for _, slowQuery := range slowQueries {
		// 分析查询
		analysis, err := qo.analyzer.AnalyzeQuery(ctx, slowQuery.Query, slowQuery.Params)
		if err != nil {
			continue
		}

		// 优化查询
		optimized, err := qo.optimizer.OptimizeQuery(analysis)
		if err != nil {
			continue
		}

		suggestion := OptimizationSuggestion{
			QueryID:        slowQuery.ID,
			OriginalQuery:  slowQuery.Query,
			OptimizedQuery: optimized.OptimizedQuery,
			Duration:       slowQuery.Duration,
			Complexity:     analysis.Complexity,
			Optimizations:  optimized.Optimizations,
			EstimatedGain:  qo.estimateGain(analysis, optimized),
			Timestamp:      slowQuery.Timestamp,
		}

		suggestions = append(suggestions, suggestion)
	}

	// 按预估收益排序
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].EstimatedGain > suggestions[j].EstimatedGain
	})

	return suggestions, nil
}

// OptimizationSuggestion 优化建议
type OptimizationSuggestion struct {
	QueryID        string        `json:"query_id"`
	OriginalQuery  string        `json:"original_query"`
	OptimizedQuery string        `json:"optimized_query"`
	Duration       time.Duration `json:"duration"`
	Complexity     int           `json:"complexity"`
	Optimizations  []string      `json:"optimizations"`
	EstimatedGain  float64       `json:"estimated_gain"`
	Timestamp      time.Time     `json:"timestamp"`
}

// estimateGain 估算性能提升
func (qo *QueryOptimizer) estimateGain(analysis *QueryAnalysis, optimized *OptimizedQuery) float64 {
	// 简化的性能提升估算
	// 实际项目中应该基于基准测试

	gain := 0.0

	// 复杂度降低
	if analysis.Complexity > 10 {
		gain += 20.0
	} else if analysis.Complexity > 5 {
		gain += 10.0
	}

	// 有优化建议
	if len(optimized.Optimizations) > 0 {
		gain += float64(len(optimized.Optimizations)) * 5.0
	}

	// 有聚合操作
	if len(analysis.Aggregations) > 0 {
		gain += 15.0
	}

	// 有 JOIN 操作
	if len(analysis.Joins) > 0 {
		gain += 10.0
	}

	return gain
}

// CreateIndexes 创建建议的索引
func (qo *QueryOptimizer) CreateIndexes(ctx context.Context, suggestions []OptimizationSuggestion) error {
	for _, suggestion := range suggestions {
		// 简化的索引创建逻辑
		// 实际项目中应该解析建议并创建相应的索引

		for _, optimization := range suggestion.Optimizations {
			if strings.Contains(optimization, "index") {
				log.Printf("Creating index for query %s: %s", suggestion.QueryID, optimization)
				// 这里应该执行 CREATE INDEX 语句
			}
		}
	}

	return nil
}

// MonitorQueries 监控查询性能
func (qo *QueryOptimizer) MonitorQueries(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qo.collectMetrics(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// collectMetrics 收集指标
func (qo *QueryOptimizer) collectMetrics(ctx context.Context) {
	stats := qo.slowQueryLog.GetSlowQueryStats()

	// 记录指标到监控系统
	for key, value := range stats {
		switch v := value.(type) {
		case int:
			qo.metricsCollector.RecordDBError("slow_query_stats", key)
		case string:
			qo.metricsCollector.RecordDBError("slow_query_stats", key)
		case time.Duration:
			qo.metricsCollector.RecordDBQuery("slow_query_duration", key, v, true)
		}
	}
}

// GetQueryStats 获取查询统计信息
func (qo *QueryOptimizer) GetQueryStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 慢查询统计
	slowQueryStats := qo.slowQueryLog.GetSlowQueryStats()
	stats["slow_queries"] = slowQueryStats

	// 数据库连接统计
	sqlDB, err := qo.db.DB()
	if err == nil {
		dbStats := sqlDB.Stats()
		stats["db_connections"] = dbStats.OpenConnections
		stats["db_in_use"] = dbStats.InUse
		stats["db_idle"] = dbStats.Idle
		stats["db_wait_count"] = dbStats.WaitCount
		stats["db_wait_duration"] = dbStats.WaitDuration
	}

	return stats
}

// detectQueryType 检测查询类型
func (qa *QueryOptimizerAnalyzer) detectQueryType(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))

	if strings.HasPrefix(query, "select") {
		return "SELECT"
	} else if strings.HasPrefix(query, "insert") {
		return "INSERT"
	} else if strings.HasPrefix(query, "update") {
		return "UPDATE"
	} else if strings.HasPrefix(query, "delete") {
		return "DELETE"
	} else if strings.HasPrefix(query, "create") {
		return "CREATE"
	} else if strings.HasPrefix(query, "drop") {
		return "DROP"
	} else if strings.HasPrefix(query, "alter") {
		return "ALTER"
	}

	return "UNKNOWN"
}

// extractTables 提取表名
func (qa *QueryOptimizerAnalyzer) extractTables(query string) []string {
	var tables []string

	// 简化的表名提取
	// 实际项目中应该使用 SQL 解析器

	if strings.Contains(query, "from") {
		parts := strings.Split(query, "from")
		if len(parts) > 1 {
			fromPart := strings.TrimSpace(parts[1])
			tableName := strings.Split(fromPart, " ")[0]
			tables = append(tables, strings.Trim(tableName, "\"`"))
		}
	}

	// JOIN 子句
	joinKeywords := []string{"join", "inner join", "left join", "right join", "full join"}
	for _, keyword := range joinKeywords {
		if strings.Contains(query, keyword) {
			parts := strings.Split(query, keyword)
			if len(parts) > 1 {
				joinPart := strings.TrimSpace(parts[1])
				tableName := strings.Split(joinPart, " ")[0]
				tables = append(tables, strings.Trim(tableName, "\"`"))
			}
		}
	}

	// 去重
	uniqueTables := make(map[string]bool)
	var result []string
	for _, table := range tables {
		if table != "" && !uniqueTables[table] {
			uniqueTables[table] = true
			result = append(result, table)
		}
	}

	return result
}

// extractFields 提取字段
func (qa *QueryOptimizerAnalyzer) extractFields(query string) []string {
	var fields []string

	// 简化的字段提取
	if strings.Contains(query, "select") {
		parts := strings.Split(query, "select")
		if len(parts) > 1 {
			selectPart := strings.TrimSpace(parts[1])

			// 找到 FROM 之前的部分
			fromIndex := strings.Index(selectPart, "from")
			if fromIndex != -1 {
				selectPart = selectPart[:fromIndex]
			}

			// 处理 SELECT *
			if strings.Contains(selectPart, "*") {
				fields = append(fields, "*")
			} else {
				// 分割字段
				fieldParts := strings.Split(selectPart, ",")
				for _, field := range fieldParts {
					field = strings.TrimSpace(field)
					// 移除函数调用
					if idx := strings.Index(field, "("); idx != -1 {
						field = field[:idx]
					}
					if field != "" {
						fields = append(fields, strings.Trim(field, "\"`"))
					}
				}
			}
		}
	}

	return fields
}

// extractWhereConditions 提取 WHERE 条件
func (qa *QueryOptimizerAnalyzer) extractWhereConditions(query string) []string {
	var conditions []string

	if strings.Contains(query, "where") {
		parts := strings.Split(query, "where")
		if len(parts) > 1 {
			wherePart := strings.TrimSpace(parts[1])

			// 找到下一个关键字之前
			keywords := []string{"order by", "group by", "limit", "having", "offset"}
			for _, keyword := range keywords {
				if idx := strings.Index(wherePart, keyword); idx != -1 {
					wherePart = wherePart[:idx]
					break
				}
			}

			// 简化处理，直接返回整个 WHERE 子句
			if wherePart != "" {
				conditions = append(conditions, strings.TrimSpace(wherePart))
			}
		}
	}

	return conditions
}

// extractJoins 提取 JOIN 信息
func (qa *QueryOptimizerAnalyzer) extractJoins(query string) []JoinInfo {
	var joins []JoinInfo

	joinKeywords := []string{"join", "inner join", "left join", "right join", "full join"}
	for _, keyword := range joinKeywords {
		if strings.Contains(query, keyword) {
			parts := strings.Split(query, keyword)
			if len(parts) > 1 {
				joinPart := strings.TrimSpace(parts[1])

				// 提取表名和条件
				tableName := strings.Split(joinPart, " ")[0]
				var condition string

				if strings.Contains(joinPart, "on") {
					onParts := strings.Split(joinPart, "on")
					if len(onParts) > 1 {
						condition = strings.TrimSpace(onParts[1])
					}
				}

				joins = append(joins, JoinInfo{
					Type:       strings.ToUpper(keyword),
					RightTable: strings.Trim(tableName, "\"`"),
					Condition:  condition,
				})
			}
		}
	}

	return joins
}

// extractAggregations 提取聚合操作
func (qa *QueryOptimizerAnalyzer) extractAggregations(query string) []string {
	var aggregations []string

	aggFunctions := []string{"count", "sum", "avg", "min", "max", "stddev", "variance"}

	for _, funcName := range aggFunctions {
		if strings.Contains(strings.ToLower(query), funcName) {
			aggregations = append(aggregations, strings.ToUpper(funcName))
		}
	}

	return aggregations
}

// extractOrderBy 提取排序字段
func (qa *QueryOptimizerAnalyzer) extractOrderBy(query string) []string {
	var orderFields []string

	if strings.Contains(query, "order by") {
		parts := strings.Split(query, "order by")
		if len(parts) > 1 {
			orderByPart := strings.TrimSpace(parts[1])

			// 找到下一个关键字之前
			keywords := []string{"limit", "offset", "having"}
			for _, keyword := range keywords {
				if idx := strings.Index(orderByPart, keyword); idx != -1 {
					orderByPart = orderByPart[:idx]
					break
				}
			}

			// 分割排序字段
			fieldParts := strings.Split(orderByPart, ",")
			for _, field := range fieldParts {
				field = strings.TrimSpace(field)
				// 移除排序方向
				if strings.HasSuffix(field, " desc") || strings.HasSuffix(field, "asc") {
					field = field[:len(field)-4]
				}
				if field != "" {
					orderFields = append(orderFields, strings.Trim(field, "\"`"))
				}
			}
		}
	}

	return orderFields
}

// extractGroupBy 提取分组字段
func (qa *QueryOptimizerAnalyzer) extractGroupBy(query string) []string {
	var groupFields []string

	if strings.Contains(query, "group by") {
		parts := strings.Split(query, "group by")
		if len(parts) > 1 {
			groupByPart := strings.TrimSpace(parts[1])

			// 找到下一个关键字之前
			keywords := []string{"order by", "limit", "offset", "having"}
			for _, keyword := range keywords {
				if idx := strings.Index(groupByPart, keyword); idx != -1 {
					groupByPart = groupByPart[:idx]
					break
				}
			}

			// 分割分组字段
			fieldParts := strings.Split(groupByPart, ",")
			for _, field := range fieldParts {
				field = strings.TrimSpace(field)
				if field != "" {
					groupFields = append(groupFields, strings.Trim(field, "\"`"))
				}
			}
		}
	}

	return groupFields
}

// extractLimit 提取 LIMIT 值
func (qa *QueryOptimizerAnalyzer) extractLimit(query string) int {
	if strings.Contains(query, "limit") {
		parts := strings.Split(query, "limit")
		if len(parts) > 1 {
			limitPart := strings.TrimSpace(parts[1])

			// 提取数字
			re := regexp.MustCompile(`\d+`)
			matches := re.FindString(limitPart)
			if matches != "" {
				if limit, err := fmt.Sscanf(matches, "%d", new(int)); err == nil {
					return limit
				}
			}
		}
	}

	return 0
}

// calculateComplexity 计算查询复杂度
func (qa *QueryOptimizerAnalyzer) calculateComplexity(analysis *QueryAnalysis) int {
	complexity := 1

	// 基础复杂度
	complexity += len(analysis.Tables)
	complexity += len(analysis.Joins) * 2
	complexity += len(analysis.WhereConditions)
	complexity += len(analysis.Aggregations) * 2

	// 特殊操作增加复杂度
	if analysis.QueryType == "SELECT" {
		if len(analysis.Aggregations) > 0 {
			complexity += 5
		}
		if len(analysis.GroupBy) > 0 {
			complexity += 3
		}
		if len(analysis.OrderBy) > 0 {
			complexity += 2
		}
	}

	return complexity
}
