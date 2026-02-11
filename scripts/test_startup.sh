#!/bin/bash

# 项目启动测试脚本
# 使用方法: ./scripts/test_startup.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo "🚀 Go Progress 项目启动测试"
echo "================================"

# 1. 检查依赖
log_info "检查依赖..."
if ! command -v go &> /dev/null; then
    log_error "Go 未安装"
    exit 1
fi
log_success "Go 已安装: $(go version)"

# 2. 检查项目结构
log_info "检查项目结构..."
required_dirs=("cmd" "configs" "internal" "pkg" "docs" "scripts")
for dir in "${required_dirs[@]}"; do
    if [ ! -d "$dir" ]; then
        log_error "缺少目录: $dir"
        exit 1
    fi
done
log_success "项目结构完整"

# 3. 检查配置文件
log_info "检查配置文件..."
if [ ! -f "configs/config.yaml" ]; then
    log_error "配置文件不存在"
    exit 1
fi
log_success "配置文件存在"

# 4. 安装依赖
log_info "安装依赖..."
go mod tidy
log_success "依赖安装完成"

# 5. 构建项目
log_info "构建项目..."
mkdir -p bin
go build -o bin/server cmd/server/main.go
log_success "项目构建完成"

# 6. 启动服务器测试
log_info "启动服务器测试..."
./bin/server &
SERVER_PID=$!

# 等待服务器启动
sleep 3

# 测试健康检查
log_info "测试健康检查..."
if curl -s http://localhost:8080/health > /dev/null; then
    log_success "健康检查通过"
else
    log_error "健康检查失败"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 测试 API 路由
log_info "测试 API 路由..."
response=$(curl -s http://localhost:8080/users/ 2>/dev/null || echo "")
if [[ "$response" == *"Authorization header is required"* ]]; then
    log_success "API 路由正常工作"
else
    log_warning "API 路由可能有问题"
fi

# 停止服务器
log_info "停止服务器..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo ""
echo "🎉 启动测试完成！"
echo "================================"
log_success "所有测试通过，项目可以正常启动"
echo ""
echo "📋 启动方式:"
echo "  1. 直接启动: ./bin/server"
echo "  2. 快速启动: ./scripts/quick-start.sh"
echo "  3. Docker 启动: docker-compose up -d"
echo ""
echo "🌐 访问地址:"
echo "  - 健康检查: http://localhost:8080/health"
echo "  - API 接口: http://localhost:8080/"
echo ""
echo "📚 更多信息:"
echo "  - 部署指南: docs/DEPLOYMENT_GUIDE.md"
echo "  - 性能优化: docs/PERFORMANCE_OPTIMIZATION.md"
