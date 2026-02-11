package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
	"user_crud_jwt/pkg/testing"
)

func main() {
	var (
		baseURL  = flag.String("url", "http://localhost:8080", "Base URL for testing")
		testType = flag.String("type", "all", "Test type: api, load, stress, benchmark, response, all")
		help     = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	fmt.Println("🚀 Go Progress 性能测试工具")
	fmt.Println("================================")

	// 检查服务器是否可用
	apiTest := testing.NewAPITest(*baseURL)
	if !checkServerHealth(apiTest) {
		log.Fatalf("❌ 服务器不可用: %s", *baseURL)
	}

	fmt.Printf("✅ 服务器可用: %s\n", *baseURL)
	fmt.Println()

	// 根据测试类型运行相应的测试
	switch *testType {
	case "api":
		runAPITests(apiTest)
	case "load":
		runLoadTests(apiTest)
	case "stress":
		runStressTests(apiTest)
	case "benchmark":
		runBenchmarkTests(apiTest)
	case "response":
		runResponseTimeTests(apiTest)
	case "all":
		runAllTests(apiTest)
	default:
		fmt.Printf("❌ 未知的测试类型: %s\n", *testType)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("用法:")
	fmt.Println("  perf_test [选项]")
	fmt.Println("")
	fmt.Println("选项:")
	fmt.Println("  -url string        测试服务器地址 (默认: http://localhost:8080)")
	fmt.Println("  -type string       测试类型 (api|load|stress|benchmark|response|all) (默认: all)")
	fmt.Println("  -concurrency int   并发数 (默认: 50)")
	fmt.Println("  -duration duration 测试时长 (默认: 30s)")
	fmt.Println("  -help              显示帮助信息")
	fmt.Println("")
	fmt.Println("测试类型说明:")
	fmt.Println("  api        - API 性能测试")
	fmt.Println("  load       - 负载测试")
	fmt.Println("  stress     - 压力测试")
	fmt.Println("  benchmark  - 基准测试")
	fmt.Println("  response   - 响应时间测试")
	fmt.Println("  all        - 运行所有测试")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  perf_test -url=http://localhost:8080 -type=api")
	fmt.Println("  perf_test -concurrency=100 -duration=60s")
	fmt.Println("  perf_test -type=stress -concurrency=200")
}

func checkServerHealth(apiTest *testing.APITest) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := apiTest.HealthCheckTest()(ctx)
	return err == nil
}

func runAPITests(apiTest *testing.APITest) {
	fmt.Println("🔧 运行 API 性能测试")
	apiTest.RunAPITests()
}

func runLoadTests(apiTest *testing.APITest) {
	fmt.Println("🔄 运行负载测试")
	apiTest.RunLoadTest()
}

func runStressTests(apiTest *testing.APITest) {
	fmt.Println("💪 运行压力测试")
	apiTest.RunStressTest()
}

func runBenchmarkTests(apiTest *testing.APITest) {
	fmt.Println("📊 运行基准测试")
	apiTest.BenchmarkEndpoints()
}

func runResponseTimeTests(apiTest *testing.APITest) {
	fmt.Println("⏱️ 运行响应时间测试")
	apiTest.TestResponseTime()
}

func runAllTests(apiTest *testing.APITest) {
	fmt.Println("🎯 运行完整性能测试套件")
	fmt.Println("================================")

	// 1. 基准测试
	fmt.Println("📊 第1阶段: 基准测试")
	apiTest.BenchmarkEndpoints()
	fmt.Println()

	// 2. 响应时间测试
	fmt.Println("⏱️ 第2阶段: 响应时间测试")
	apiTest.TestResponseTime()
	fmt.Println()

	// 3. API 性能测试
	fmt.Println("🚀 第3阶段: API 性能测试")
	apiTest.RunAPITests()
	fmt.Println()

	// 4. 负载测试
	fmt.Println("🔄 第4阶段: 负载测试")
	apiTest.RunLoadTest()
	fmt.Println()

	// 5. 压力测试
	fmt.Println("💪 第5阶段: 压力测试")
	apiTest.RunStressTest()
	fmt.Println()

	fmt.Println("🎉 完整性能测试套件执行完成！")
	fmt.Println("================================")
	fmt.Println("📝 建议:")
	fmt.Println("1. 查看 P95 响应时间，确保 < 100ms")
	fmt.Println("2. 检查错误率，确保 < 0.1%")
	fmt.Println("3. 监控 QPS，评估系统吞吐量")
	fmt.Println("4. 根据压力测试结果确定最大并发数")
	fmt.Println("5. 使用基准测试结果优化关键路径")
}
