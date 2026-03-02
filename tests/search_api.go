package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword string `json:"keyword"`
	Type    string `json:"type"`
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Mobile string `json:"mobile"`
	Code   string `json:"code"`
}

const SEARCH_BASE_URL = "http://localhost:8080"

func main() {
	fmt.Println("=== Moment 搜索功能测试开始 ===")
	
	// 1. 登录获取token
	token, err := login()
	if err != nil {
		fmt.Printf("❌ 登录失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 登录成功，token: %s...\n", token[:20])
	
	// 2. 测试搜索动态
	testSearchMoments(token)
	
	// 3. 测试搜索话题
	testSearchTopics()
	
	// 4. 测试获取热门话题
	testGetHotTopics()
	
	// 5. 测试获取用户动态
	testGetUserMoments(token)
	
	fmt.Println("=== Moment 搜索功能测试完成 ===")
}

func login() (string, error) {
	loginReq := LoginRequest{
		Mobile: "13800138000",
		Code:   "123456",
	}
	
	jsonData, _ := json.Marshal(loginReq)
	resp, err := http.Post(SEARCH_BASE_URL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	if result["code"].(float64) != 0 {
		return "", fmt.Errorf("login failed: %v", result["message"])
	}
	
	return result["data"].(string), nil
}

func testSearchMoments(token string) {
	fmt.Println("\n--- 测试搜索动态 ---")
	
	searchReq := SearchRequest{
		Keyword: "测试",
		Type:    "all",
		Page:    1,
		Limit:   10,
	}
	
	jsonData, _ := json.Marshal(searchReq)
	req, _ := http.NewRequest("POST", SEARCH_BASE_URL+"/search/moments", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 搜索动态失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	if result["code"].(float64) == 0 {
		data := result["data"].(map[string]interface{})
		posts := data["posts"].([]interface{})
		fmt.Printf("✅ 搜索动态成功，找到 %d 条结果\n", len(posts))
	} else {
		fmt.Printf("❌ 搜索动态失败: %v\n", result["message"])
	}
}

func testSearchTopics() {
	fmt.Println("\n--- 测试搜索话题 ---")
	
	resp, err := http.Get(SEARCH_BASE_URL + "/search/topics?keyword=生活&limit=10")
	if err != nil {
		fmt.Printf("❌ 搜索话题失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	if result["code"].(float64) == 0 {
		topics := result["data"].([]interface{})
		fmt.Printf("✅ 搜索话题成功，找到 %d 个话题\n", len(topics))
	} else {
		fmt.Printf("❌ 搜索话题失败: %v\n", result["message"])
	}
}

func testGetHotTopics() {
	fmt.Println("\n--- 测试获取热门话题 ---")
	
	resp, err := http.Get(SEARCH_BASE_URL + "/search/hot-topics?limit=5")
	if err != nil {
		fmt.Printf("❌ 获取热门话题失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	if result["code"].(float64) == 0 {
		topics := result["data"].([]interface{})
		fmt.Printf("✅ 获取热门话题成功，共 %d 个话题\n", len(topics))
	} else {
		fmt.Printf("❌ 获取热门话题失败: %v\n", result["message"])
	}
}

func testGetUserMoments(token string) {
	fmt.Println("\n--- 测试获取用户动态 ---")
	
	req, _ := http.NewRequest("GET", SEARCH_BASE_URL+"/search/users/bf5906aa-f1f7-4b6a-8abc-2af95de74a75/moments?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 获取用户动态失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	if result["code"].(float64) == 0 {
		data := result["data"].(map[string]interface{})
		posts := data["posts"].([]interface{})
		fmt.Printf("✅ 获取用户动态成功，共 %d 条动态\n", len(posts))
	} else {
		fmt.Printf("❌ 获取用户动态失败: %v\n", result["message"])
	}
}
