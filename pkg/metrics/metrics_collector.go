package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	// HTTP 指标
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// 数据库指标
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec
	dbQueryTotal        *prometheus.CounterVec
	dbErrorsTotal       *prometheus.CounterVec

	// 缓存指标
	cacheHitsTotal         *prometheus.CounterVec
	cacheMissesTotal       *prometheus.CounterVec
	cacheOperationDuration *prometheus.HistogramVec

	// 应用指标
	activeGoroutines prometheus.Gauge
	memoryUsage      prometheus.Gauge
	customMetrics    map[string]prometheus.Metric
	mu               sync.RWMutex
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// HTTP 指标
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),

		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "endpoint"},
		),

		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "endpoint"},
		),

		// 数据库指标
		dbConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		),

		dbConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
			},
		),

		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),

		dbQueryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),

		dbErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"operation", "error_type"},
		),

		// 缓存指标
		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type", "key_prefix"},
		),

		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type", "key_prefix"},
		),

		cacheOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cache_operation_duration_seconds",
				Help:    "Cache operation duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
			[]string{"operation", "cache_type"},
		),

		// 应用指标
		activeGoroutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_goroutines",
				Help: "Number of active goroutines",
			},
		),

		memoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),

		customMetrics: make(map[string]prometheus.Metric),
	}
}

// RecordHTTPRequest 记录 HTTP 请求指标
func (m *MetricsCollector) RecordHTTPRequest(method, endpoint, status string, duration time.Duration, requestSize, responseSize int) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	m.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordDBQuery 记录数据库查询指标
func (m *MetricsCollector) RecordDBQuery(operation, table string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	m.dbQueryTotal.WithLabelValues(operation, table, status).Inc()
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

	if !success {
		m.dbErrorsTotal.WithLabelValues(operation, "query_error").Inc()
	}
}

// RecordCacheOperation 记录缓存操作指标
func (m *MetricsCollector) RecordCacheOperation(operation, cacheType, keyPrefix string, duration time.Duration, hit bool) {
	if hit {
		m.cacheHitsTotal.WithLabelValues(cacheType, keyPrefix).Inc()
	} else {
		m.cacheMissesTotal.WithLabelValues(cacheType, keyPrefix).Inc()
	}

	m.cacheOperationDuration.WithLabelValues(operation, cacheType).Observe(duration.Seconds())
}

// UpdateDBConnections 更新数据库连接指标
func (m *MetricsCollector) UpdateDBConnections(active, idle int) {
	m.dbConnectionsActive.Set(float64(active))
	m.dbConnectionsIdle.Set(float64(idle))
}

// UpdateActiveGoroutines 更新活跃 goroutine 数量
func (m *MetricsCollector) UpdateActiveGoroutines(count int) {
	m.activeGoroutines.Set(float64(count))
}

// UpdateMemoryUsage 更新内存使用量
func (m *MetricsCollector) UpdateMemoryUsage(bytes int) {
	m.memoryUsage.Set(float64(bytes))
}

// RecordDBError 记录数据库错误
func (m *MetricsCollector) RecordDBError(operation, errorType string) {
	m.dbErrorsTotal.WithLabelValues(operation, errorType).Inc()
}

// UpdateSystemMetrics 更新系统指标
func (m *MetricsCollector) UpdateSystemMetrics() {
	// 这里可以添加更多的系统指标收集
	// 例如内存使用、CPU 使用等
}

// AddCustomMetric 添加自定义指标
func (m *MetricsCollector) AddCustomMetric(name string, metric prometheus.Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customMetrics[name] = metric
}

// GetCustomMetric 获取自定义指标
func (m *MetricsCollector) GetCustomMetric(name string) (prometheus.Metric, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metric, exists := m.customMetrics[name]
	return metric, exists
}

// PerformanceTracker 性能跟踪器
type PerformanceTracker struct {
	startTime time.Time
	operation string
	labels    []string
	collector *MetricsCollector
}

// NewPerformanceTracker 创建性能跟踪器
func NewPerformanceTracker(collector *MetricsCollector, operation string, labels ...string) *PerformanceTracker {
	return &PerformanceTracker{
		startTime: time.Now(),
		operation: operation,
		labels:    labels,
		collector: collector,
	}
}

// Finish 完成跟踪
func (pt *PerformanceTracker) Finish() {
	duration := time.Since(pt.startTime)

	switch pt.operation {
	case "http_request":
		if len(pt.labels) >= 3 {
			method, endpoint, status := pt.labels[0], pt.labels[1], pt.labels[2]
			pt.collector.RecordHTTPRequest(method, endpoint, status, duration, 0, 0)
		}
	case "db_query":
		if len(pt.labels) >= 2 {
			operation, table := pt.labels[0], pt.labels[1]
			pt.collector.RecordDBQuery(operation, table, duration, true)
		}
	case "cache_operation":
		if len(pt.labels) >= 3 {
			operation, cacheType, hit := pt.labels[0], pt.labels[1], pt.labels[2]
			isHit := hit == "hit"
			pt.collector.RecordCacheOperation(operation, cacheType, "", duration, isHit)
		}
	}
}

// ContextTracker 上下文跟踪器
type ContextTracker struct {
	context.Context
	collector *MetricsCollector
	startTime time.Time
	operation string
	done      chan struct{}
}

// WithPerformanceTracking 为上下文添加性能跟踪
func WithPerformanceTracking(ctx context.Context, collector *MetricsCollector, operation string) context.Context {
	return &ContextTracker{
		Context:   ctx,
		collector: collector,
		startTime: time.Now(),
		operation: operation,
		done:      make(chan struct{}),
	}
}

// Done 返回完成通道
func (ct *ContextTracker) Done() <-chan struct{} {
	return ct.done
}

// finish 完成上下文跟踪
func (ct *ContextTracker) finish() {
	duration := time.Since(ct.startTime)

	// 记录操作完成时间
	switch ct.operation {
	case "request":
		ct.collector.httpRequestDuration.WithLabelValues("", "").Observe(duration.Seconds())
	case "database":
		ct.collector.dbQueryDuration.WithLabelValues("", "").Observe(duration.Seconds())
	case "cache":
		ct.collector.cacheOperationDuration.WithLabelValues("", "").Observe(duration.Seconds())
	}

	close(ct.done)
}

// MetricsMiddleware 指标中间件助手
type MetricsMiddleware struct {
	collector *MetricsCollector
}

// NewMetricsMiddleware 创建指标中间件
func NewMetricsMiddleware(collector *MetricsCollector) *MetricsMiddleware {
	return &MetricsMiddleware{collector: collector}
}

// TrackRequest 跟踪请求
func (m *MetricsMiddleware) TrackRequest(method, endpoint string) func(status int, requestSize, responseSize int) {
	start := time.Now()

	return func(status int, requestSize, responseSize int) {
		duration := time.Since(start)
		statusStr := getStatusCategory(status)
		m.collector.RecordHTTPRequest(method, endpoint, statusStr, duration, requestSize, responseSize)
	}
}

// getStatusCategory 获取状态分类
func getStatusCategory(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// 全局指标收集器实例
var GlobalCollector *MetricsCollector

// InitMetrics 初始化全局指标收集器
func InitMetrics() {
	GlobalCollector = NewMetricsCollector()
}

// GetGlobalCollector 获取全局指标收集器
func GetGlobalCollector() *MetricsCollector {
	if GlobalCollector == nil {
		InitMetrics()
	}
	return GlobalCollector
}
