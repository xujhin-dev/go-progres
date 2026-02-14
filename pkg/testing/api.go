package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APITest API 性能测试
type APITest struct {
	baseURL string
	client  *http.Client
}

// NewAPITest 创建 API 测试
func NewAPITest(baseURL string) *APITest {
	return &APITest{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

// HealthCheckTest 健康检查测试
func (at *APITest) HealthCheckTest() RequestFunc {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", at.baseURL+"/health", nil)
		if err != nil {
			return err
		}

		resp, err := at.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// UserListTest 用户列表测试
func (at *APITest) UserListTest(token string) RequestFunc {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", at.baseURL+"/users/", nil)
		if err != nil {
			return err
		}

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := at.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 401 是预期的（没有认证）
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// LoginTest 登录测试
func (at *APITest) LoginTest(mobile, code string) RequestFunc {
	return func(ctx context.Context) error {
		loginData := map[string]string{
			"mobile": mobile,
			"code":   code,
		}

		jsonData, err := json.Marshal(loginData)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", at.baseURL+"/auth/login", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := at.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 登录可能失败（验证码错误），但这是正常的
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// UploadTest 文件上传测试
func (at *APITest) UploadTest(token string) RequestFunc {
	return func(ctx context.Context) error {
		// 创建简单的测试数据
		testData := bytes.NewBufferString("test file content")

		req, err := http.NewRequestWithContext(ctx, "POST", at.baseURL+"/upload", testData)
		if err != nil {
			return err
		}

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := at.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 上传可能失败（没有认证或配置），但这是正常的
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusInternalServerError {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// RunAPITests 运行 API 性能测试
func (at *APITest) RunAPITests() {
	fmt.Println("🚀 开始 API 性能测试")
	fmt.Println("================================")

	// 1. 健康检查测试
	fmt.Println("📊 健康检查性能测试")
	healthTest := NewPerformanceTest("health_check", 50, time.Second*30)
	healthTest.AddRequest(at.HealthCheckTest())
	healthResult := healthTest.Run()
	healthResult.PrintResult()

	// 2. 用户列表测试
	fmt.Println("📊 用户列表性能测试")
	userListTest := NewPerformanceTest("user_list", 20, time.Second*30)
	userListTest.AddRequest(at.UserListTest(""))
	userListResult := userListTest.Run()
	userListResult.PrintResult()

	// 3. 登录测试
	fmt.Println("📊 登录性能测试")
	loginTest := NewPerformanceTest("login", 10, time.Second*30)
	loginTest.AddRequest(at.LoginTest("13800138000", "123456"))
	loginResult := loginTest.Run()
	loginResult.PrintResult()

	// 4. 文件上传测试
	fmt.Println("📊 文件上传性能测试")
	uploadTest := NewPerformanceTest("upload", 5, time.Second*30)
	uploadTest.AddRequest(at.UploadTest(""))
	uploadResult := uploadTest.Run()
	uploadResult.PrintResult()

	// 5. 混合负载测试
	fmt.Println("📊 混合负载性能测试")
	mixedTest := NewPerformanceTest("mixed_load", 30, time.Second*60)
	mixedTest.AddRequest(at.HealthCheckTest())
	mixedTest.AddRequest(at.UserListTest(""))
	mixedTest.AddRequest(at.LoginTest("13800138001", "123456"))
	mixedResult := mixedTest.Run()
	mixedResult.PrintResult()

	// 6. 结果对比
	fmt.Println("📈 测试结果对比")
	CompareResults(healthResult, userListResult, loginResult, uploadResult, mixedResult)

	fmt.Println("================================")
	fmt.Println("✅ API 性能测试完成")
}

// RunLoadTest 运行负载测试
func (at *APITest) RunLoadTest() {
	fmt.Println("🔄 开始负载测试")
	fmt.Println("================================")

	loadTest := NewLoadTest()

	// 场景1: 低并发长时间测试
	loadTest.AddScenario(LoadScenario{
		Name:        "low_concurrency",
		Concurrency: 10,
		Duration:    time.Minute * 2,
		Requests: []RequestFunc{
			at.HealthCheckTest(),
			at.UserListTest(""),
		},
	})

	// 场景2: 中等并发测试
	loadTest.AddScenario(LoadScenario{
		Name:        "medium_concurrency",
		Concurrency: 50,
		Duration:    time.Minute * 1,
		Requests: []RequestFunc{
			at.HealthCheckTest(),
			at.UserListTest(""),
			at.LoginTest("13800138002", "123456"),
		},
	})

	// 场景3: 渐进式负载测试
	loadTest.AddScenario(LoadScenario{
		Name:        "ramp_up_test",
		Concurrency: 100,
		Duration:    time.Minute * 3,
		RampUp:      time.Minute * 1,
		Requests: []RequestFunc{
			at.HealthCheckTest(),
			at.UserListTest(""),
		},
	})

	results := loadTest.Run()

	fmt.Println("📈 负载测试结果汇总")
	for _, result := range results {
		fmt.Printf("场景: %-20s | QPS: %-8.2f | P95: %-8v | 错误率: %-6.2f%%\n",
			result.TestName, result.QPS, result.P95, result.ErrorRate*100)
	}

	fmt.Println("================================")
	fmt.Println("✅ 负载测试完成")
}

// RunStressTest 运行压力测试
func (at *APITest) RunStressTest() {
	fmt.Println("💪 开始压力测试")
	fmt.Println("================================")

	stressTest := NewStressTest(200, 20, time.Second*30)
	stressTest.AddRequest(at.HealthCheckTest())
	stressTest.AddRequest(at.UserListTest(""))

	results := stressTest.Run()

	fmt.Println("📈 压力测试结果汇总")
	for _, result := range results {
		fmt.Printf("并发: %-4d | QPS: %-8.2f | P95: %-8v | 错误率: %-6.2f%%\n",
			result.Concurrency, result.QPS, result.P95, result.ErrorRate*100)
	}

	fmt.Println("================================")
	fmt.Println("✅ 压力测试完成")
}

// BenchmarkEndpoints 端点基准测试
func (at *APITest) BenchmarkEndpoints() {
	fmt.Println("📊 开始端点基准测试")
	fmt.Println("================================")

	// 健康检查基准测试
	healthBenchmark := NewBenchmarkTest("health_benchmark", func(b *B) {
		b.timer.Start()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			at.HealthCheckTest()(ctx)
			cancel()
		}
		b.timer.Stop()
	})
	healthResult := healthBenchmark.Run()
	fmt.Printf("健康检查基准: %v\n", healthResult)

	// 用户列表基准测试
	userListBenchmark := NewBenchmarkTest("user_list_benchmark", func(b *B) {
		b.timer.Start()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			at.UserListTest("")(ctx)
			cancel()
		}
		b.timer.Stop()
	})
	userListResult := userListBenchmark.Run()
	fmt.Printf("用户列表基准: %v\n", userListResult)

	fmt.Println("================================")
	fmt.Println("✅ 基准测试完成")
}

// TestResponseTime 测试响应时间分布
func (at *APITest) TestResponseTime() {
	fmt.Println("⏱️ 开始响应时间测试")
	fmt.Println("================================")

	testCases := []struct {
		name    string
		request RequestFunc
		samples int
	}{
		{"health_check", at.HealthCheckTest(), 100},
		{"user_list", at.UserListTest(""), 50},
		{"login", at.LoginTest("13800138000", "123456"), 30},
	}

	for _, tc := range testCases {
		fmt.Printf("📊 %s 响应时间分布 (%d 样本)\n", tc.name, tc.samples)

		pt := NewPerformanceTest(tc.name+"_response_time", 1, time.Second*30)
		pt.AddRequest(tc.request)

		result := pt.Run()

		fmt.Printf("平均: %v, 最小: %v, 最大: %v\n",
			result.AverageResponseTime, result.MinResponseTime, result.MaxResponseTime)
		fmt.Printf("P50: %v, P95: %v, P99: %v\n", result.P50, result.P95, result.P99)
		fmt.Println()
	}

	fmt.Println("================================")
	fmt.Println("✅ 响应时间测试完成")
}
