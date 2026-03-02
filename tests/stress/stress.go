package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	STRESS_BASE_URL = "http://localhost:8080"
	// 测试配置
	CONCURRENT_USERS  = 50
	REQUESTS_PER_USER = 10
	TEST_DURATION     = 30 * time.Second
)

type StressOTPRequest struct {
	Mobile string `json:"mobile"`
}

type StressLoginInput struct {
	Mobile string `json:"mobile"`
	Code   string `json:"code"`
}

type StressMomentPost struct {
	Content   string   `json:"content"`
	MediaURLs []string `json:"mediaUrls"`
	Type      string   `json:"type"`
	Topics    []string `json:"topics"`
}

type StressTestResult struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	TotalDuration   time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	AvgResponseTime time.Duration
	Errors          map[string]int
}

func runStressTests() {
	fmt.Println("=== Moment API 压力测试开始 ===")

	// 1. 基础功能测试
	fmt.Println("\n1. 基础功能测试...")
	stressBasicTest()

	// 2. 并发登录测试
	fmt.Println("\n2. 并发登录测试...")
	stressConcurrentLoginTest()

	// 3. 并发发布动态测试
	fmt.Println("\n3. 并发发布动态测试...")
	stressConcurrentPublishTest()

	// 4. 并发读取测试
	fmt.Println("\n4. 并发读取测试...")
	stressConcurrentReadTest()

	// 5. 混合压力测试
	fmt.Println("\n5. 混合压力测试...")
	stressMixedTest()

	fmt.Println("\n=== Moment API 压力测试完成 ===")
}

func stressBasicTest() {
	// 测试单个用户的基本操作
	token := stressLoginAndGetToken("13800139001")
	if token == "" {
		fmt.Println("❌ 基础登录测试失败")
		return
	}

	momentID := stressPublishMoment(token)
	if momentID == "" {
		fmt.Println("❌ 基础发布动态测试失败")
		return
	}

	fmt.Println("✅ 基础功能测试通过")
}

func stressConcurrentLoginTest() {
	var wg sync.WaitGroup
	results := make(chan StressTestResult, CONCURRENT_USERS)

	start := time.Now()

	for i := 0; i < CONCURRENT_USERS; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			mobile := fmt.Sprintf("1380013%04d", userID)
			result := StressTestResult{
				Errors: make(map[string]int),
			}

			requestStart := time.Now()

			token := stressLoginAndGetToken(mobile)
			if token != "" {
				result.SuccessRequests++
			} else {
				result.FailedRequests++
				result.Errors["login_failed"]++
			}

			result.TotalRequests = 1
			result.TotalDuration = time.Since(requestStart)
			result.MinResponseTime = result.TotalDuration
			result.MaxResponseTime = result.TotalDuration
			result.AvgResponseTime = result.TotalDuration

			results <- result
		}(i)
	}

	wg.Wait()
	close(results)

	// 汇总结果
	totalResult := stressAggregateResults(results)
	totalDuration := time.Since(start)

	fmt.Printf("并发登录测试结果:\n")
	fmt.Printf("  并发用户数: %d\n", CONCURRENT_USERS)
	fmt.Printf("  总耗时: %v\n", totalDuration)
	fmt.Printf("  成功请求: %d\n", totalResult.SuccessRequests)
	fmt.Printf("  失败请求: %d\n", totalResult.FailedRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(totalResult.SuccessRequests)/float64(totalResult.TotalRequests)*100)
	fmt.Printf("  平均响应时间: %v\n", totalResult.AvgResponseTime)
}

func stressConcurrentPublishTest() {
	// 先获取一些用户token
	var tokens []string
	for i := 0; i < CONCURRENT_USERS; i++ {
		mobile := fmt.Sprintf("1380014%04d", i)
		token := stressLoginAndGetToken(mobile)
		if token != "" {
			tokens = append(tokens, token)
		}
	}

	if len(tokens) == 0 {
		fmt.Println("❌ 无法获取用户token，跳过发布动态测试")
		return
	}

	var wg sync.WaitGroup
	results := make(chan StressTestResult, len(tokens)*REQUESTS_PER_USER)

	start := time.Now()

	for i, token := range tokens {
		for j := 0; j < REQUESTS_PER_USER; j++ {
			wg.Add(1)
			go func(userIndex, reqIndex int, userToken string) {
				defer wg.Done()

				result := StressTestResult{
					Errors: make(map[string]int),
				}

				requestStart := time.Now()

				momentID := stressPublishMoment(userToken)
				duration := time.Since(requestStart)

				result.TotalRequests = 1
				result.TotalDuration = duration
				result.MinResponseTime = duration
				result.MaxResponseTime = duration
				result.AvgResponseTime = duration

				if momentID != "" {
					result.SuccessRequests++
				} else {
					result.FailedRequests++
					result.Errors["publish_failed"]++
				}

				results <- result
			}(i, j, token)
		}
	}

	wg.Wait()
	close(results)

	totalResult := stressAggregateResults(results)
	totalDuration := time.Since(start)

	fmt.Printf("并发发布动态测试结果:\n")
	fmt.Printf("  并发用户数: %d\n", len(tokens))
	fmt.Printf("  每用户请求数: %d\n", REQUESTS_PER_USER)
	fmt.Printf("  总请求数: %d\n", totalResult.TotalRequests)
	fmt.Printf("  总耗时: %v\n", totalDuration)
	fmt.Printf("  成功请求: %d\n", totalResult.SuccessRequests)
	fmt.Printf("  失败请求: %d\n", totalResult.FailedRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(totalResult.SuccessRequests)/float64(totalResult.TotalRequests)*100)
	fmt.Printf("  QPS: %.2f\n", float64(totalResult.TotalRequests)/totalDuration.Seconds())
	fmt.Printf("  平均响应时间: %v\n", totalResult.AvgResponseTime)
}

func stressConcurrentReadTest() {
	// 先发布一些动态
	token := stressLoginAndGetToken("13800139002")
	if token == "" {
		fmt.Println("❌ 无法获取token，跳过读取测试")
		return
	}

	// 发布一些测试动态
	for i := 0; i < 10; i++ {
		stressPublishMoment(token)
		time.Sleep(100 * time.Millisecond)
	}

	var wg sync.WaitGroup
	results := make(chan StressTestResult, CONCURRENT_USERS*10)

	start := time.Now()

	// 并发读取动态列表
	for i := 0; i < CONCURRENT_USERS*10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result := StressTestResult{
				Errors: make(map[string]int),
			}

			requestStart := time.Now()

			success := stressGetFeed(token)
			duration := time.Since(requestStart)

			result.TotalRequests = 1
			result.TotalDuration = duration
			result.MinResponseTime = duration
			result.MaxResponseTime = duration
			result.AvgResponseTime = duration

			if success {
				result.SuccessRequests++
			} else {
				result.FailedRequests++
				result.Errors["read_failed"]++
			}

			results <- result
		}()
	}

	wg.Wait()
	close(results)

	totalResult := stressAggregateResults(results)
	totalDuration := time.Since(start)

	fmt.Printf("并发读取测试结果:\n")
	fmt.Printf("  并发读取数: %d\n", CONCURRENT_USERS*10)
	fmt.Printf("  总耗时: %v\n", totalDuration)
	fmt.Printf("  成功请求: %d\n", totalResult.SuccessRequests)
	fmt.Printf("  失败请求: %d\n", totalResult.FailedRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(totalResult.SuccessRequests)/float64(totalResult.TotalRequests)*100)
	fmt.Printf("  QPS: %.2f\n", float64(totalResult.TotalRequests)/totalDuration.Seconds())
	fmt.Printf("  平均响应时间: %v\n", totalResult.AvgResponseTime)
}

func stressMixedTest() {
	// 获取多个用户token
	var tokens []string
	for i := 0; i < 20; i++ {
		mobile := fmt.Sprintf("1380015%04d", i)
		token := stressLoginAndGetToken(mobile)
		if token != "" {
			tokens = append(tokens, token)
		}
	}

	if len(tokens) == 0 {
		fmt.Println("❌ 无法获取用户token，跳过混合压力测试")
		return
	}

	var wg sync.WaitGroup
	results := make(chan StressTestResult, 0)
	stopChan := make(chan bool)

	start := time.Now()
	requestCount := 0
	var countMutex sync.Mutex

	// 启动不同类型的并发请求
	// 1. 发布动态
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					requestStart := time.Now()
					momentID := stressPublishMoment(token)
					duration := time.Since(requestStart)

					result := StressTestResult{
						TotalRequests:   1,
						TotalDuration:   duration,
						MinResponseTime: duration,
						MaxResponseTime: duration,
						AvgResponseTime: duration,
						Errors:          make(map[string]int),
					}

					if momentID != "" {
						result.SuccessRequests++
					} else {
						result.FailedRequests++
						result.Errors["publish_failed"]++
					}

					results <- result
					countMutex.Lock()
					requestCount++
					countMutex.Unlock()

					time.Sleep(200 * time.Millisecond)
				}
			}
		}(tokens[i%len(tokens)])
	}

	// 2. 读取动态列表
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					requestStart := time.Now()
					success := stressGetFeed(token)
					duration := time.Since(requestStart)

					result := StressTestResult{
						TotalRequests:   1,
						TotalDuration:   duration,
						MinResponseTime: duration,
						MaxResponseTime: duration,
						AvgResponseTime: duration,
						Errors:          make(map[string]int),
					}

					if success {
						result.SuccessRequests++
					} else {
						result.FailedRequests++
						result.Errors["read_failed"]++
					}

					results <- result
					countMutex.Lock()
					requestCount++
					countMutex.Unlock()

					time.Sleep(100 * time.Millisecond)
				}
			}
		}(tokens[i%len(tokens)])
	}

	// 3. 点赞操作
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					requestStart := time.Now()
					success := stressToggleLike(token, "test-moment-id")
					duration := time.Since(requestStart)

					result := StressTestResult{
						TotalRequests:   1,
						TotalDuration:   duration,
						MinResponseTime: duration,
						MaxResponseTime: duration,
						AvgResponseTime: duration,
						Errors:          make(map[string]int),
					}

					if success {
						result.SuccessRequests++
					} else {
						result.FailedRequests++
						result.Errors["like_failed"]++
					}

					results <- result
					countMutex.Lock()
					requestCount++
					countMutex.Unlock()

					time.Sleep(300 * time.Millisecond)
				}
			}
		}(tokens[i%len(tokens)])
	}

	// 收集结果
	var allResults []StressTestResult
	go func() {
		for result := range results {
			allResults = append(allResults, result)
		}
	}()

	// 运行指定时间后停止
	time.Sleep(TEST_DURATION)
	close(stopChan)

	wg.Wait()
	close(results)

	totalDuration := time.Since(start)

	// 计算总结果
	totalResult := StressTestResult{Errors: make(map[string]int)}
	for _, result := range allResults {
		totalResult.TotalRequests += result.TotalRequests
		totalResult.SuccessRequests += result.SuccessRequests
		totalResult.FailedRequests += result.FailedRequests

		for errType, count := range result.Errors {
			totalResult.Errors[errType] += count
		}
	}

	if totalResult.TotalRequests > 0 {
		totalResult.AvgResponseTime = totalDuration / time.Duration(totalResult.TotalRequests)
	}

	fmt.Printf("混合压力测试结果:\n")
	fmt.Printf("  测试时长: %v\n", totalDuration)
	fmt.Printf("  总请求数: %d\n", totalResult.TotalRequests)
	fmt.Printf("  成功请求: %d\n", totalResult.SuccessRequests)
	fmt.Printf("  失败请求: %d\n", totalResult.FailedRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(totalResult.SuccessRequests)/float64(totalResult.TotalRequests)*100)
	fmt.Printf("  QPS: %.2f\n", float64(totalResult.TotalRequests)/totalDuration.Seconds())
	fmt.Printf("  平均响应时间: %v\n", totalResult.AvgResponseTime)

	// 显示错误统计
	if len(totalResult.Errors) > 0 {
		fmt.Printf("  错误统计:\n")
		for errType, count := range totalResult.Errors {
			fmt.Printf("    %s: %d\n", errType, count)
		}
	}
}

func stressLoginAndGetToken(mobile string) string {
	// 发送验证码
	otpReq := StressOTPRequest{Mobile: mobile}
	jsonData, _ := json.Marshal(otpReq)
	resp, err := http.Post(STRESS_BASE_URL+"/auth/otp", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// 登录
	loginData := StressLoginInput{
		Mobile: mobile,
		Code:   "123456",
	}

	jsonData, _ = json.Marshal(loginData)
	resp, err = http.Post(STRESS_BASE_URL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
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

func stressPublishMoment(token string) string {
	moment := StressMomentPost{
		Content:   fmt.Sprintf("压力测试动态 - %s", time.Now().Format("15:04:05")),
		MediaURLs: []string{"test.jpg"},
		Type:      "text",
		Topics:    []string{"压力测试"},
	}

	jsonData, _ := json.Marshal(moment)
	req, _ := http.NewRequest("POST", STRESS_BASE_URL+"/moments/publish", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return "success"
	}
	return ""
}

func stressGetFeed(token string) bool {
	req, _ := http.NewRequest("GET", STRESS_BASE_URL+"/moments/feed?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func stressToggleLike(token, momentID string) bool {
	likeReq := map[string]interface{}{
		"targetId":   momentID,
		"targetType": "post",
	}

	jsonData, _ := json.Marshal(likeReq)
	req, _ := http.NewRequest("POST", STRESS_BASE_URL+"/moments/like", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func stressAggregateResults(results <-chan StressTestResult) StressTestResult {
	var total StressTestResult
	total.Errors = make(map[string]int)

	for result := range results {
		total.TotalRequests += result.TotalRequests
		total.SuccessRequests += result.SuccessRequests
		total.FailedRequests += result.FailedRequests

		if total.MinResponseTime == 0 || result.MinResponseTime < total.MinResponseTime {
			total.MinResponseTime = result.MinResponseTime
		}

		if result.MaxResponseTime > total.MaxResponseTime {
			total.MaxResponseTime = result.MaxResponseTime
		}

		// 累计错误
		for errType, count := range result.Errors {
			total.Errors[errType] += count
		}
	}

	if total.TotalRequests > 0 {
		total.AvgResponseTime = total.TotalDuration / time.Duration(total.TotalRequests)
	}

	return total
}
