package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	CONCURRENT_BASE_URL = "http://localhost:8080"
	// 并发测试配置
	MAX_CONCURRENT_USERS = 100
	BURST_REQUESTS       = 1000
	LATENCY_SAMPLE_SIZE  = 1000
)

type ConcurrentMetrics struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalLatency      int64 // 纳秒
	MinLatency        int64
	MaxLatency        int64
	ActiveConnections int64
	PeakConnections   int64
}

type ConcurrentOTPRequest struct {
	Mobile string `json:"mobile"`
}

type ConcurrentLoginInput struct {
	Mobile string `json:"mobile"`
	Code   string `json:"code"`
}

type ConcurrentMomentPost struct {
	Content   string   `json:"content"`
	MediaURLs []string `json:"mediaUrls"`
	Type      string   `json:"type"`
	Topics    []string `json:"topics"`
}

func runConcurrentTests() {
	fmt.Println("=== Moment API 高级并发测试开始 ===")

	// 1. 突发并发测试
	fmt.Println("\n1. 突发并发测试...")
	burstConcurrentTest()

	// 2. 渐进式并发测试
	fmt.Println("\n2. 渐进式并发测试...")
	progressiveConcurrentTest()

	// 3. 持续高并发测试
	fmt.Println("\n3. 持续高并发测试...")
	sustainedConcurrentTest()

	// 4. 连接池测试
	fmt.Println("\n4. 连接池测试...")
	connectionPoolTest()

	// 5. 资源使用监控测试
	fmt.Println("\n5. 资源使用监控测试...")
	resourceMonitoringTest()

	fmt.Println("\n=== Moment API 高级并发测试完成 ===")
}

func burstConcurrentTest() {
	var metrics ConcurrentMetrics
	metrics.MinLatency = int64(^uint64(0) >> 1) // 最大值

	var wg sync.WaitGroup
	start := time.Now()

	// 突发发送大量请求
	for i := 0; i < BURST_REQUESTS; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			atomic.AddInt64(&metrics.ActiveConnections, 1)
			current := atomic.LoadInt64(&metrics.ActiveConnections)
			if current > atomic.LoadInt64(&metrics.PeakConnections) {
				atomic.StoreInt64(&metrics.PeakConnections, current)
			}
			defer atomic.AddInt64(&metrics.ActiveConnections, -1)

			requestStart := time.Now()

			// 执行请求
			mobile := fmt.Sprintf("1380016%04d", requestID%MAX_CONCURRENT_USERS)
			token := concurrentLoginAndGetToken(mobile)

			latency := time.Since(requestStart).Nanoseconds()
			atomic.AddInt64(&metrics.TotalRequests, 1)
			atomic.AddInt64(&metrics.TotalLatency, latency)

			// 更新最小/最大延迟
			for {
				currentMin := atomic.LoadInt64(&metrics.MinLatency)
				if latency >= currentMin || atomic.CompareAndSwapInt64(&metrics.MinLatency, currentMin, latency) {
					break
				}
			}

			for {
				currentMax := atomic.LoadInt64(&metrics.MaxLatency)
				if latency <= currentMax || atomic.CompareAndSwapInt64(&metrics.MaxLatency, currentMax, latency) {
					break
				}
			}

			if token != "" {
				atomic.AddInt64(&metrics.SuccessRequests, 1)
			} else {
				atomic.AddInt64(&metrics.FailedRequests, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("突发并发测试结果:\n")
	fmt.Printf("  突发请求数: %d\n", BURST_REQUESTS)
	fmt.Printf("  总耗时: %v\n", duration)
	fmt.Printf("  成功请求: %d\n", metrics.SuccessRequests)
	fmt.Printf("  失败请求: %d\n", metrics.FailedRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(metrics.SuccessRequests)/float64(metrics.TotalRequests)*100)
	fmt.Printf("  QPS: %.2f\n", float64(metrics.TotalRequests)/duration.Seconds())
	fmt.Printf("  平均延迟: %.2fms\n", float64(metrics.TotalLatency)/float64(metrics.TotalRequests)/1000000)
	fmt.Printf("  最小延迟: %.2fms\n", float64(metrics.MinLatency)/1000000)
	fmt.Printf("  最大延迟: %.2fms\n", float64(metrics.MaxLatency)/1000000)
	fmt.Printf("  峰值连接数: %d\n", metrics.PeakConnections)
}

func progressiveConcurrentTest() {
	userCounts := []int{10, 25, 50, 75, 100}

	for _, userCount := range userCounts {
		fmt.Printf("  测试 %d 并发用户...\n", userCount)

		var metrics ConcurrentMetrics
		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < userCount; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				mobile := fmt.Sprintf("1380017%04d", userID)
				token := concurrentLoginAndGetToken(mobile)

				if token != "" {
					atomic.AddInt64(&metrics.SuccessRequests, 1)
				} else {
					atomic.AddInt64(&metrics.FailedRequests, 1)
				}
				atomic.AddInt64(&metrics.TotalRequests, 1)
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		fmt.Printf("    %d用户: 耗时%v, 成功率%.2f%%, QPS%.2f\n",
			userCount, duration,
			float64(metrics.SuccessRequests)/float64(metrics.TotalRequests)*100,
			float64(metrics.TotalRequests)/duration.Seconds())

		// 等待一段时间再进行下一轮测试
		time.Sleep(1 * time.Second)
	}
}

func sustainedConcurrentTest() {
	const TEST_DURATION = 10 * time.Second
	const WORKER_COUNT = 20

	var metrics ConcurrentMetrics
	var wg sync.WaitGroup
	stopChan := make(chan bool)

	start := time.Now()

	// 启动工作协程
	for i := 0; i < WORKER_COUNT; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			mobile := fmt.Sprintf("1380018%04d", workerID)
			token := concurrentLoginAndGetToken(mobile)

			if token == "" {
				return
			}

			for {
				select {
				case <-stopChan:
					return
				default:
					requestStart := time.Now()

					// 执行各种操作
					switch workerID % 3 {
					case 0:
						concurrentPublishMoment(token)
					case 1:
						concurrentGetFeed(token)
					case 2:
						concurrentToggleLike(token, "test-moment")
					}

					latency := time.Since(requestStart)
					atomic.AddInt64(&metrics.TotalRequests, 1)
					atomic.AddInt64(&metrics.TotalLatency, latency.Nanoseconds())

					if latency.Nanoseconds() < atomic.LoadInt64(&metrics.MinLatency) || metrics.MinLatency == 0 {
						atomic.StoreInt64(&metrics.MinLatency, latency.Nanoseconds())
					}

					if latency.Nanoseconds() > atomic.LoadInt64(&metrics.MaxLatency) {
						atomic.StoreInt64(&metrics.MaxLatency, latency.Nanoseconds())
					}

					atomic.AddInt64(&metrics.SuccessRequests, 1)

					// 控制请求频率
					time.Sleep(50 * time.Millisecond)
				}
			}
		}(i)
	}

	// 运行指定时间
	time.Sleep(TEST_DURATION)
	close(stopChan)

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("持续高并发测试结果:\n")
	fmt.Printf("  测试时长: %v\n", duration)
	fmt.Printf("  工作协程数: %d\n", WORKER_COUNT)
	fmt.Printf("  总请求数: %d\n", metrics.TotalRequests)
	fmt.Printf("  成功请求: %d\n", metrics.SuccessRequests)
	fmt.Printf("  QPS: %.2f\n", float64(metrics.TotalRequests)/duration.Seconds())
	fmt.Printf("  平均延迟: %.2fms\n", float64(metrics.TotalLatency)/float64(metrics.TotalRequests)/1000000)
}

func connectionPoolTest() {
	const CONNECTION_COUNT = 50
	const REQUESTS_PER_CONNECTION = 20

	var wg sync.WaitGroup
	var totalRequests int64
	var successRequests int64
	start := time.Now()

	for i := 0; i < CONNECTION_COUNT; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			mobile := fmt.Sprintf("1380019%04d", connID)
			token := concurrentLoginAndGetToken(mobile)

			if token == "" {
				return
			}

			for j := 0; j < REQUESTS_PER_CONNECTION; j++ {
				if concurrentGetFeed(token) {
					atomic.AddInt64(&successRequests, 1)
				}
				atomic.AddInt64(&totalRequests, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("连接池测试结果:\n")
	fmt.Printf("  连接数: %d\n", CONNECTION_COUNT)
	fmt.Printf("  每连接请求数: %d\n", REQUESTS_PER_CONNECTION)
	fmt.Printf("  总请求数: %d\n", totalRequests)
	fmt.Printf("  成功请求: %d\n", successRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(successRequests)/float64(totalRequests)*100)
	fmt.Printf("  总耗时: %v\n", duration)
	fmt.Printf("  QPS: %.2f\n", float64(totalRequests)/duration.Seconds())
}

func resourceMonitoringTest() {
	const MONITORING_DURATION = 15 * time.Second

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var wg sync.WaitGroup
	stopChan := make(chan bool)
	requestCount := int64(0)

	// 启动请求协程
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			mobile := fmt.Sprintf("1380020%04d", workerID)
			token := concurrentLoginAndGetToken(mobile)

			if token == "" {
				return
			}

			for {
				select {
				case <-stopChan:
					return
				default:
					concurrentGetFeed(token)
					atomic.AddInt64(&requestCount, 1)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}(i)
	}

	// 监控系统资源
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var samples []struct {
		timestamp  time.Time
		alloc      uint64
		goroutines int
		requests   int64
	}

	// 运行指定时间后停止
	time.Sleep(MONITORING_DURATION)
	close(stopChan)

	wg.Wait()
	runtime.ReadMemStats(&m2)

	fmt.Printf("资源使用监控测试结果:\n")
	fmt.Printf("  测试时长: %v\n", MONITORING_DURATION)
	fmt.Printf("  总请求数: %d\n", requestCount)
	fmt.Printf("  QPS: %.2f\n", float64(requestCount)/MONITORING_DURATION.Seconds())
	fmt.Printf("  内存使用:\n")
	fmt.Printf("    开始: %.2f MB\n", float64(m1.Alloc)/1024/1024)
	fmt.Printf("    结束: %.2f MB\n", float64(m2.Alloc)/1024/1024)
	fmt.Printf("    增长: %.2f MB\n", float64(m2.Alloc-m1.Alloc)/1024/1024)
	fmt.Printf("  Goroutine数:\n")
	fmt.Printf("    开始: %d\n", runtime.NumGoroutine())
	fmt.Printf("    结束: %d\n", runtime.NumGoroutine())

	if len(samples) > 0 {
		var maxAlloc uint64
		var maxGoroutines int
		for _, sample := range samples {
			if sample.alloc > maxAlloc {
				maxAlloc = sample.alloc
			}
			if sample.goroutines > maxGoroutines {
				maxGoroutines = sample.goroutines
			}
		}

		fmt.Printf("    峰值内存: %.2f MB\n", float64(maxAlloc)/1024/1024)
		fmt.Printf("    峰值Goroutine: %d\n", maxGoroutines)
	}
}

func concurrentLoginAndGetToken(mobile string) string {
	otpReq := ConcurrentOTPRequest{Mobile: mobile}
	jsonData, _ := json.Marshal(otpReq)
	resp, err := http.Post(CONCURRENT_BASE_URL+"/auth/otp", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	loginData := ConcurrentLoginInput{
		Mobile: mobile,
		Code:   "123456",
	}

	jsonData, _ = json.Marshal(loginData)
	resp, err = http.Post(CONCURRENT_BASE_URL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if token, ok := result["data"].(string); ok {
		return token
	}
	return ""
}

func concurrentPublishMoment(token string) bool {
	moment := ConcurrentMomentPost{
		Content:   fmt.Sprintf("并发测试动态 - %s", time.Now().Format("15:04:05")),
		MediaURLs: []string{"test.jpg"},
		Type:      "text",
		Topics:    []string{"并发测试"},
	}

	jsonData, _ := json.Marshal(moment)
	req, _ := http.NewRequest("POST", CONCURRENT_BASE_URL+"/moments/publish", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func concurrentGetFeed(token string) bool {
	req, _ := http.NewRequest("GET", CONCURRENT_BASE_URL+"/moments/feed?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func concurrentToggleLike(token, momentID string) bool {
	likeReq := map[string]interface{}{
		"targetId":   momentID,
		"targetType": "post",
	}

	jsonData, _ := json.Marshal(likeReq)
	req, _ := http.NewRequest("POST", CONCURRENT_BASE_URL+"/moments/like", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
