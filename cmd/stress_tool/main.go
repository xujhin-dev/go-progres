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

// Config
const (
	BaseURL     = "http://localhost:8080"
	TotalUsers  = 10000 // 模拟 10000 个用户并发
	TotalStock  = 5     // 优惠券只有 5 张
)

var (
	TestCouponID int
	httpClient   *http.Client
)

func init() {
	// 优化 HTTP Client 配置
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 2000
	t.MaxIdleConnsPerHost = 2000
	t.MaxConnsPerHost = 2000
	httpClient = &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
	}
}

func main() {
	// 1. 创建优惠券 (管理员操作)
	createCoupon()

	fmt.Printf("开始压测：模拟 %d 个用户抢 %d 张券 (CouponID: %d)...\n", TotalUsers, TotalStock, TestCouponID)
	time.Sleep(1 * time.Second)

	// 2. 并发抢券
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	start := time.Now()

	for i := 1; i <= TotalUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			success := claimCoupon(userID)
			mu.Lock()
			if success {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	qps := float64(TotalUsers) / duration.Seconds()

	fmt.Println("--------------------------------------------------")
	fmt.Printf("压测结束，耗时: %v\n", duration)
	fmt.Printf("总请求数: %d\n", TotalUsers)
	fmt.Printf("QPS: %.2f\n", qps)
	fmt.Printf("成功抢到: %d (预期: %d)\n", successCount, TotalStock)
	fmt.Printf("抢券失败: %d\n", failCount)
	fmt.Println("--------------------------------------------------")
}

func createCoupon() {
	url := fmt.Sprintf("%s/coupons/create_test", BaseURL)
	payload := map[string]interface{}{
		"name":       "压测专用券",
		"total":      TotalStock,
		"amount":     100.0,
		"start_time": time.Now().Format(time.RFC3339),
		"end_time":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("创建优惠券失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应以获取 ID
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("创建优惠券响应: %s\n", string(respBody))

	var result struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}
	TestCouponID = result.Data.ID
}

func claimCoupon(userID int) bool {
	// 使用测试后门接口，直接传 user_id
	url := fmt.Sprintf("%s/coupons/%d/claim_test?user_id=%d", BaseURL, TestCouponID, userID)
	resp, err := httpClient.Post(url, "application/json", nil)
	if err != nil {
		// fmt.Printf("User %d 请求失败: %v\n", userID, err)
		return false
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != 200 {
		return false
	}

	// 检查业务状态码
	var result struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false
	}

	return result.Code == 0
}
