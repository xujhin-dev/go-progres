#!/bin/bash

# API 测试脚本
BASE_URL="http://localhost:8080"

echo "=========================================="
echo "🚀 API 功能测试"
echo "=========================================="
echo ""

# 1. 健康检查
echo "1️⃣  测试健康检查..."
curl -s $BASE_URL/health | jq .
echo ""

# 2. 注册用户
echo "2️⃣  测试用户注册..."
REGISTER_RESULT=$(curl -s -X POST $BASE_URL/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"demo_user","password":"demo123456","email":"demo@example.com"}')
echo $REGISTER_RESULT | jq .
echo ""

# 3. 登录获取 Token
echo "3️⃣  测试用户登录..."
LOGIN_RESULT=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"demo_user","password":"demo123456"}')
echo $LOGIN_RESULT | jq .

TOKEN=$(echo $LOGIN_RESULT | jq -r '.data.token')
echo "Token: $TOKEN"
echo ""

# 4. 获取用户列表
echo "4️⃣  测试获取用户列表（需要认证）..."
curl -s -X GET "$BASE_URL/users/?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

# 5. 测试 TraceID
echo "5️⃣  测试 TraceID 追踪..."
curl -v $BASE_URL/health 2>&1 | grep -i "x-trace-id"
echo ""

# 6. 测试权限控制（尝试修改其他用户）
echo "6️⃣  测试权限控制（应该失败）..."
curl -s -X PUT $BASE_URL/users/999 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"hacker"}' | jq .
echo ""

# 7. 测试修改自己的信息
echo "7️⃣  测试修改自己的信息（应该成功）..."
curl -s -X PUT $BASE_URL/users/2 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"demo_user_updated","email":"demo_updated@example.com"}' | jq .
echo ""

# 8. 测试修改密码
echo "8️⃣  测试修改密码..."
curl -s -X PUT $BASE_URL/users/password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"demo123456","new_password":"newpass123456"}' | jq .
echo ""

# 9. 测试用新密码登录
echo "9️⃣  测试用新密码登录..."
curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"demo_user","password":"newpass123456"}' | jq .
echo ""

echo "=========================================="
echo "✅ 测试完成！"
echo "=========================================="
echo ""
echo "📊 可用的接口："
echo "  - Swagger 文档: $BASE_URL/swagger/index.html"
echo "  - 健康检查: $BASE_URL/health"
echo "  - Prometheus 指标: $BASE_URL/metrics"
echo ""
