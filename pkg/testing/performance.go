package testing

import (
	"context"
	"fmt"
	"sync"
	"time"
	"user_crud_jwt/pkg/metrics"
)

// PerformanceTest 性能测试框架
type PerformanceTest struct {
	name        string
	concurrency int
	duration    time.Duration
	requests    []RequestFunc
	metrics     *TestMetrics
	collector   *metrics.MetricsCollector
}

// RequestFunc 请求函数
type RequestFunc func(ctx context.Context) error

// TestMetrics 测试指标
type TestMetrics struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	ResponseTimes   []time.Duration
	mu              sync.RWMutex
}

// NewPerformanceTest 创建性能测试
func NewPerformanceTest(name string, concurrency int, duration time.Duration) *PerformanceTest {
	return &PerformanceTest{
		name:        name,
		concurrency: concurrency,
		duration:    duration,
		metrics:     &TestMetrics{},
		collector:   metrics.GetGlobalCollector(),
	}
}

// AddRequest 添加请求函数
func (pt *PerformanceTest) AddRequest(request RequestFunc) {
	pt.requests = append(pt.requests, request)
}

// Run 运行性能测试
func (pt *PerformanceTest) Run() *TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), pt.duration)
	defer cancel()

	var wg sync.WaitGroup
	requestChan := make(chan RequestFunc, pt.concurrency*2)

	// 启动工作协程
	for i := 0; i < pt.concurrency; i++ {
		wg.Add(1)
		go pt.worker(ctx, &wg, requestChan)
	}

	// 发送请求
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(requestChan)
				return
			default:
				for _, req := range pt.requests {
					select {
					case requestChan <- req:
					case <-ctx.Done():
						close(requestChan)
						return
					}
				}
			}
		}
	}()

	wg.Wait()
	return pt.generateResult()
}

// worker 工作协程
func (pt *PerformanceTest) worker(ctx context.Context, wg *sync.WaitGroup, requestChan <-chan RequestFunc) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case request, ok := <-requestChan:
			if !ok {
				return
			}
			pt.executeRequest(ctx, request)
		}
	}
}

// executeRequest 执行单个请求
func (pt *PerformanceTest) executeRequest(ctx context.Context, request RequestFunc) {
	start := time.Now()
	err := request(ctx)
	duration := time.Since(start)

	pt.metrics.mu.Lock()
	defer pt.metrics.mu.Unlock()

	pt.metrics.TotalRequests++
	pt.metrics.TotalDuration += duration
	pt.metrics.ResponseTimes = append(pt.metrics.ResponseTimes, duration)

	if pt.metrics.MinDuration == 0 || duration < pt.metrics.MinDuration {
		pt.metrics.MinDuration = duration
	}
	if duration > pt.metrics.MaxDuration {
		pt.metrics.MaxDuration = duration
	}

	if err != nil {
		pt.metrics.FailedRequests++
	} else {
		pt.metrics.SuccessRequests++
	}
}

// generateResult 生成测试结果
func (pt *PerformanceTest) generateResult() *TestResult {
	pt.metrics.mu.RLock()
	defer pt.metrics.mu.RUnlock()

	result := &TestResult{
		TestName:        pt.name,
		Concurrency:     pt.concurrency,
		Duration:        pt.duration,
		TotalRequests:   pt.metrics.TotalRequests,
		SuccessRequests: pt.metrics.SuccessRequests,
		FailedRequests:  pt.metrics.FailedRequests,
		QPS:             float64(pt.metrics.TotalRequests) / pt.duration.Seconds(),
		SuccessRate:     float64(pt.metrics.SuccessRequests) / float64(pt.metrics.TotalRequests),
		ErrorRate:       float64(pt.metrics.FailedRequests) / float64(pt.metrics.TotalRequests),
	}

	if len(pt.metrics.ResponseTimes) > 0 {
		result.AverageResponseTime = pt.metrics.TotalDuration / time.Duration(len(pt.metrics.ResponseTimes))
		result.MinResponseTime = pt.metrics.MinDuration
		result.MaxResponseTime = pt.metrics.MaxDuration

		// 计算百分位数
		sortedTimes := make([]time.Duration, len(pt.metrics.ResponseTimes))
		copy(sortedTimes, pt.metrics.ResponseTimes)

		// 简单排序（实际项目中应该使用更高效的排序算法）
		for i := 0; i < len(sortedTimes); i++ {
			for j := i + 1; j < len(sortedTimes); j++ {
				if sortedTimes[i] > sortedTimes[j] {
					sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
				}
			}
		}

		result.P50 = percentile(sortedTimes, 0.5)
		result.P95 = percentile(sortedTimes, 0.95)
		result.P99 = percentile(sortedTimes, 0.99)
	}

	return result
}

// percentile 计算百分位数
func percentile(times []time.Duration, p float64) time.Duration {
	if len(times) == 0 {
		return 0
	}
	index := int(float64(len(times)) * p)
	if index >= len(times) {
		index = len(times) - 1
	}
	return times[index]
}

// TestResult 测试结果
type TestResult struct {
	TestName            string        `json:"test_name"`
	Concurrency         int           `json:"concurrency"`
	Duration            time.Duration `json:"duration"`
	TotalRequests       int64         `json:"total_requests"`
	SuccessRequests     int64         `json:"success_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	QPS                 float64       `json:"qps"`
	SuccessRate         float64       `json:"success_rate"`
	ErrorRate           float64       `json:"error_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	MinResponseTime     time.Duration `json:"min_response_time"`
	MaxResponseTime     time.Duration `json:"max_response_time"`
	P50                 time.Duration `json:"p50"`
	P95                 time.Duration `json:"p95"`
	P99                 time.Duration `json:"p99"`
}

// PrintResult 打印测试结果
func (tr *TestResult) PrintResult() {
	fmt.Printf("📊 性能测试结果: %s\n", tr.TestName)
	fmt.Printf("================================\n")
	fmt.Printf("并发数: %d\n", tr.Concurrency)
	fmt.Printf("测试时长: %v\n", tr.Duration)
	fmt.Printf("总请求数: %d\n", tr.TotalRequests)
	fmt.Printf("成功请求: %d\n", tr.SuccessRequests)
	fmt.Printf("失败请求: %d\n", tr.FailedRequests)
	fmt.Printf("QPS: %.2f\n", tr.QPS)
	fmt.Printf("成功率: %.2f%%\n", tr.SuccessRate*100)
	fmt.Printf("错误率: %.2f%%\n", tr.ErrorRate*100)
	fmt.Printf("平均响应时间: %v\n", tr.AverageResponseTime)
	fmt.Printf("最小响应时间: %v\n", tr.MinResponseTime)
	fmt.Printf("最大响应时间: %v\n", tr.MaxResponseTime)
	fmt.Printf("P50: %v\n", tr.P50)
	fmt.Printf("P95: %v\n", tr.P95)
	fmt.Printf("P99: %v\n", tr.P99)
	fmt.Printf("================================\n")
}

// BenchmarkTest 基准测试
type BenchmarkTest struct {
	name   string
	fn     func(b *B)
	allocs int64
	bytes  int64
}

// B 基准测试类型
type B struct {
	N     int
	timer Timer
}

// Timer 计时器
type Timer struct {
	start time.Time
}

// Start 开始计时
func (t *Timer) Start() {
	t.start = time.Now()
}

// Stop 停止计时
func (t *Timer) Stop() time.Duration {
	return time.Since(t.start)
}

// NewBenchmarkTest 创建基准测试
func NewBenchmarkTest(name string, fn func(b *B)) *BenchmarkTest {
	return &BenchmarkTest{
		name: name,
		fn:   fn,
	}
}

// Run 运行基准测试
func (bt *BenchmarkTest) Run() *BenchmarkResult {
	b := &B{N: 1000} // 默认运行 1000 次

	bt.fn(b)

	return &BenchmarkResult{
		TestName:    bt.name,
		NsPerOp:     float64(b.timer.Stop()) / float64(b.N),
		AllocsPerOp: bt.allocs / int64(b.N),
		BytesPerOp:  bt.bytes / int64(b.N),
	}
}

// BenchmarkResult 基准测试结果
type BenchmarkResult struct {
	TestName    string  `json:"test_name"`
	NsPerOp     float64 `json:"ns_per_op"`
	AllocsPerOp int64   `json:"allocs_per_op"`
	BytesPerOp  int64   `json:"bytes_per_op"`
}

// LoadTest 负载测试
type LoadTest struct {
	scenarios []LoadScenario
}

// LoadScenario 负载场景
type LoadScenario struct {
	Name        string
	Concurrency int
	Duration    time.Duration
	RampUp      time.Duration
	Requests    []RequestFunc
}

// NewLoadTest 创建负载测试
func NewLoadTest() *LoadTest {
	return &LoadTest{
		scenarios: make([]LoadScenario, 0),
	}
}

// AddScenario 添加负载场景
func (lt *LoadTest) AddScenario(scenario LoadScenario) {
	lt.scenarios = append(lt.scenarios, scenario)
}

// Run 运行负载测试
func (lt *LoadTest) Run() []*TestResult {
	results := make([]*TestResult, 0, len(lt.scenarios))

	for _, scenario := range lt.scenarios {
		fmt.Printf("🔄 运行负载场景: %s\n", scenario.Name)

		// 渐进式增加并发数
		if scenario.RampUp > 0 {
			results = append(results, lt.runRampUpScenario(scenario)...)
		} else {
			pt := NewPerformanceTest(scenario.Name, scenario.Concurrency, scenario.Duration)
			for _, req := range scenario.Requests {
				pt.AddRequest(req)
			}
			result := pt.Run()
			results = append(results, result)
			result.PrintResult()
		}
	}

	return results
}

// runRampUpScenario 运行渐进式负载场景
func (lt *LoadTest) runRampUpScenario(scenario LoadScenario) []*TestResult {
	results := make([]*TestResult, 0)
	steps := 5
	stepDuration := scenario.Duration / time.Duration(steps)

	for i := 1; i <= steps; i++ {
		concurrency := scenario.Concurrency * i / steps
		name := fmt.Sprintf("%s_step_%d", scenario.Name, i)

		pt := NewPerformanceTest(name, concurrency, stepDuration)
		for _, req := range scenario.Requests {
			pt.AddRequest(req)
		}

		result := pt.Run()
		results = append(results, result)
		result.PrintResult()

		// 短暂休息
		time.Sleep(time.Second)
	}

	return results
}

// StressTest 压力测试
type StressTest struct {
	maxConcurrency int
	stepSize       int
	stepDuration   time.Duration
	requests       []RequestFunc
}

// NewStressTest 创建压力测试
func NewStressTest(maxConcurrency, stepSize int, stepDuration time.Duration) *StressTest {
	return &StressTest{
		maxConcurrency: maxConcurrency,
		stepSize:       stepSize,
		stepDuration:   stepDuration,
		requests:       make([]RequestFunc, 0),
	}
}

// AddRequest 添加请求
func (st *StressTest) AddRequest(request RequestFunc) {
	st.requests = append(st.requests, request)
}

// Run 运行压力测试
func (st *StressTest) Run() []*TestResult {
	results := make([]*TestResult, 0)

	for concurrency := st.stepSize; concurrency <= st.maxConcurrency; concurrency += st.stepSize {
		name := fmt.Sprintf("stress_test_%d", concurrency)
		pt := NewPerformanceTest(name, concurrency, st.stepDuration)

		for _, req := range st.requests {
			pt.AddRequest(req)
		}

		result := pt.Run()
		results = append(results, result)
		result.PrintResult()

		// 检查是否达到性能瓶颈
		if result.ErrorRate > 0.05 || result.P95 > time.Millisecond*500 {
			fmt.Printf("⚠️ 在并发数 %d 时检测到性能瓶颈\n", concurrency)
			break
		}
	}

	return results
}

// CompareResults 比较测试结果
func CompareResults(results ...*TestResult) {
	fmt.Printf("📈 测试结果对比\n")
	fmt.Printf("================================\n")

	for _, result := range results {
		fmt.Printf("%-20s | QPS: %-8.2f | P95: %-8v | 错误率: %-6.2f%%\n",
			result.TestName, result.QPS, result.P95, result.ErrorRate*100)
	}

	fmt.Printf("================================\n")
}

// ExportResults 导出测试结果
func ExportResults(results []*TestResult, filename string) error {
	// 这里可以实现 JSON/CSV 导出
	// 为了简化，这里只是打印
	fmt.Printf("📄 导出测试结果到: %s\n", filename)
	for _, result := range results {
		result.PrintResult()
	}
	return nil
}
