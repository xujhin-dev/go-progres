package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OTPRequest struct {
	Mobile string `json:"mobile"`
}

type LoginInput struct {
	Mobile string `json:"mobile"`
	Code   string `json:"code"`
}

type MomentPost struct {
	Content   string   `json:"content"`
	MediaURLs []string `json:"mediaUrls"`
	Type      string   `json:"type"`
	Topics    []string `json:"topics"`
}

type Comment struct {
	Content string `json:"content"`
}

type AuditRequest struct {
	Status string `json:"status"`
}

type LikeRequest struct {
	TargetID   string `json:"targetId"`
	TargetType string `json:"targetType"`
}

const BASE_URL = "http://localhost:8080"

func main() {
	fmt.Println("=== Moment API 功能测试开始 ===")

	// 1. 注册并登录用户
	token := loginAndGetToken()
	if token == "" {
		fmt.Println("❌ 用户登录失败，测试终止")
		return
	}
	fmt.Println("✅ 用户登录成功")

	// 2. 发布动态
	momentID := publishMoment(token)
	if momentID == "" {
		fmt.Println("❌ 发布动态失败，测试终止")
		return
	}
	fmt.Printf("✅ 发布动态成功，ID: %s\n", momentID)

	// 3. 获取动态列表
	getFeed(token)

	// 4. 审核动态（管理员功能）
	auditPost(token, momentID)

	// 5. 添加评论
	addComment(token, momentID)

	// 6. 获取评论列表
	getComments(token, momentID)

	// 7. 点赞/取消点赞
	toggleLike(token, momentID)

	// 8. 获取话题列表
	getTopics(token)

	fmt.Println("=== Moment API 功能测试完成 ===")
}

func loginAndGetToken() string {
	mobile := "13800138000"

	// 1. 发送验证码
	otpReq := OTPRequest{
		Mobile: mobile,
	}

	jsonData, _ := json.Marshal(otpReq)
	resp, err := http.Post(BASE_URL+"/auth/otp", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("发送验证码请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// 2. 使用验证码登录（假设验证码是123456）
	loginData := LoginInput{
		Mobile: mobile,
		Code:   "123456", // 这是测试用的默认验证码
	}

	jsonData, _ = json.Marshal(loginData)
	resp, err = http.Post(BASE_URL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("登录请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("登录响应: %s\n", string(body))

	if token, ok := result["data"].(string); ok {
		return token
	}
	return ""
}

func auditPost(token string, momentID string) {
	auditReq := AuditRequest{
		Status: "approved", // 审核通过
	}

	jsonData, _ := json.Marshal(auditReq)
	url := fmt.Sprintf("%s/moments/%s/audit", BASE_URL, momentID)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("审核动态失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ 审核动态成功: %v\n", result["message"])
}

func publishMoment(token string) string {
	moment := MomentPost{
		Content:   "这是一条测试动态，包含了丰富的内容描述。",
		MediaURLs: []string{"image1.jpg", "image2.jpg"},
		Type:      "image",
		Topics:    []string{"生活", "分享"},
	}

	jsonData, _ := json.Marshal(moment)
	req, _ := http.NewRequest("POST", BASE_URL+"/moments/publish", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("发布动态请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("发布动态响应: %s\n", string(body))

	if data, ok := result["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok && id != "" {
			return id
		}
		// 如果ID为空，尝试从用户动态列表中获取最新的
		return getLatestMomentID(token)
	}
	return ""
}

func getLatestMomentID(token string) string {
	req, _ := http.NewRequest("GET", BASE_URL+"/moments/feed?page=1&limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if data, ok := result["data"].(map[string]interface{}); ok {
		if items, ok := data["items"].([]interface{}); ok && len(items) > 0 {
			if item, ok := items[0].(map[string]interface{}); ok {
				if id, ok := item["id"].(string); ok {
					return id
				}
			}
		}
	}
	return ""
}

func getFeed(token string) {
	req, _ := http.NewRequest("GET", BASE_URL+"/moments/feed?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("获取动态列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ 获取动态列表成功: %v\n", result["message"])
}

func addComment(token string, momentID string) {
	comment := Comment{
		Content: "这是一条测试评论",
	}

	jsonData, _ := json.Marshal(comment)
	url := fmt.Sprintf("%s/moments/%s/comments", BASE_URL, momentID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("添加评论失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ 添加评论成功: %v\n", result["message"])
}

func getComments(token string, momentID string) {
	url := fmt.Sprintf("%s/moments/%s/comments?page=1&limit=10", BASE_URL, momentID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("获取评论列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ 获取评论列表成功: %v\n", result["message"])
}

func toggleLike(token string, momentID string) {
	likeReq := LikeRequest{
		TargetID:   momentID,
		TargetType: "post",
	}

	jsonData, _ := json.Marshal(likeReq)
	req, _ := http.NewRequest("POST", BASE_URL+"/moments/like", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("点赞操作失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ 点赞操作成功: %v\n", result["message"])
}

func getTopics(token string) {
	req, _ := http.NewRequest("GET", BASE_URL+"/moments/topics", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("获取话题列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ 获取话题列表成功: %v\n", result["message"])
}
