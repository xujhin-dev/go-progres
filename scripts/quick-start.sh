#!/bin/bash

# 快速启动脚本 - 一键设置和部署项目
# 使用方法: ./scripts/quick-start.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 项目信息
PROJECT_NAME="Go Progress"
PROJECT_VERSION="1.0.0"

# 打印带颜色的标题
print_title() {
    echo -e "${PURPLE}"
    echo "=========================================="
    echo "🚀 $PROJECT_NAME v$PROJECT_VERSION"
    echo "=========================================="
    echo -e "${NC}"
}

# 打印步骤
print_step() {
    echo -e "${BLUE}📋 $1${NC}"
}

# 打印成功
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# 打印警告
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 打印错误
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查系统要求
check_requirements() {
    print_step "检查系统要求..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装，请先安装 Docker"
        echo "安装命令: curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose 未安装，请先安装 Docker Compose"
        echo "安装命令: sudo curl -L \"https://github.com/docker/compose/releases/latest/download/docker-compose-\$(uname -s)-\$(uname -m)\" -o /usr/local/bin/docker-compose"
        exit 1
    fi
    
    # 检查 Git
    if ! command -v git &> /dev/null; then
        print_error "Git 未安装，请先安装 Git"
        echo "安装命令: sudo apt-get install git (Ubuntu/Debian)"
        exit 1
    fi
    
    print_success "系统要求检查通过"
}

# 检查端口占用
check_ports() {
    print_step "检查端口占用..."
    
    ports=(80 443 8080 5432 6379 3000 9090)
    occupied_ports=()
    
    for port in "${ports[@]}"; do
        if lsof -i :$port &> /dev/null; then
            occupied_ports+=($port)
        fi
    done
    
    if [ ${#occupied_ports[@]} -gt 0 ]; then
        print_warning "以下端口被占用: ${occupied_ports[*]}"
        echo "请确保这些端口可用，或修改配置文件中的端口设置"
        read -p "是否继续? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        print_success "所有端口可用"
    fi
}

# 设置环境
setup_environment() {
    print_step "设置环境配置..."
    
    # 选择环境
    echo "请选择部署环境:"
    echo "1) development (开发环境)"
    echo "2) testing (测试环境)"
    echo "3) production (生产环境)"
    read -p "请输入选择 [1-3]: " -n 1 -r
    echo
    
    case $REPLY in
        1)
            ENV="development"
            ;;
        2)
            ENV="testing"
            ;;
        3)
            ENV="production"
            ;;
        *)
            print_error "无效选择"
            exit 1
            ;;
    esac
    
    # 设置环境
    ./scripts/setup-env.sh $ENV
    
    print_success "环境配置完成: $ENV"
}

# 配置环境变量
configure_env() {
    print_step "配置环境变量..."
    
    if [ ! -f .env ]; then
        print_error ".env 文件不存在"
        exit 1
    fi
    
    if [ "$ENV" = "production" ]; then
        print_warning "生产环境需要手动配置以下重要参数:"
        echo ""
        echo "🔐 必须修改的配置:"
        echo "  - DATABASE_PASSWORD (数据库密码)"
        echo "  - REDIS_PASSWORD (Redis 密码)"
        echo "  - JWT_SECRET (JWT 密钥)"
        echo "  - GITHUB_REPOSITORY (GitHub 仓库)"
        echo ""
        echo "📋 可选配置:"
        echo "  - OSS 配置 (文件上传)"
        echo "  - 支付配置 (支付宝/微信)"
        echo "  - 通知配置 (Slack/邮件)"
        echo ""
        
        read -p "是否现在编辑配置文件? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            ${EDITOR:-vim} .env
        fi
    fi
    
    print_success "环境变量配置完成"
}

# 构建和部署
deploy_project() {
    print_step "构建和部署项目..."
    
    # 拉取最新代码
    if [ -d ".git" ]; then
        git pull origin main 2>/dev/null || print_warning "无法拉取最新代码"
    fi
    
    # 运行部署脚本
    ./scripts/deploy.sh $ENV latest
    
    print_success "项目部署完成"
}

# 验证部署
verify_deployment() {
    print_step "验证部署..."
    
    # 等待服务启动
    echo "等待服务启动..."
    sleep 30
    
    # 健康检查
    max_attempts=10
    attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f http://localhost:8080/health &> /dev/null; then
            print_success "应用健康检查通过"
            break
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            print_error "健康检查失败"
            echo "请检查服务状态: docker-compose -f docker-compose.prod.yml ps"
            echo "查看日志: docker-compose -f docker-compose.prod.yml logs app"
            exit 1
        fi
        
        echo "健康检查失败，重试中... ($attempt/$max_attempts)"
        sleep 10
        ((attempt++))
    done
}

# 显示访问信息
show_access_info() {
    print_step "部署成功！访问信息:"
    echo ""
    echo -e "${CYAN}🌐 应用服务:${NC}"
    echo "  - API 接口: http://localhost:8080"
    echo "  - 健康检查: http://localhost:8080/health"
    echo "  - API 文档: http://localhost:8080/swagger/index.html"
    echo ""
    echo -e "${CYAN}📊 监控服务:${NC}"
    echo "  - Grafana: http://localhost:3000 (admin/admin)"
    echo "  - Prometheus: http://localhost:9090"
    echo ""
    echo -e "${CYAN}🔧 管理命令:${NC}"
    echo "  - 查看服务状态: docker-compose -f docker-compose.prod.yml ps"
    echo "  - 查看日志: docker-compose -f docker-compose.prod.yml logs -f"
    echo "  - 停止服务: docker-compose -f docker-compose.prod.yml down"
    echo "  - 重启服务: docker-compose -f docker-compose.prod.yml restart"
    echo ""
    echo -e "${CYAN}📚 更多信息:${NC}"
    echo "  - 部署文档: docs/DEPLOYMENT_GUIDE.md"
    echo "  - API 测试: ./scripts/test_api.sh"
    echo ""
}

# 主函数
main() {
    print_title
    
    echo -e "${CYAN}欢迎使用 $PROJECT_NAME 快速启动脚本！${NC}"
    echo ""
    echo "这个脚本将帮助您："
    echo "✓ 检查系统要求"
    echo "✓ 设置环境配置"
    echo "✓ 部署项目"
    echo "✓ 验证部署结果"
    echo ""
    
    read -p "是否继续? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "取消启动"
        exit 0
    fi
    
    echo ""
    
    # 执行所有步骤
    check_requirements
    check_ports
    setup_environment
    configure_env
    deploy_project
    verify_deployment
    show_access_info
    
    echo ""
    print_success "🎉 快速启动完成！"
    echo -e "${CYAN}如果遇到问题，请查看 docs/DEPLOYMENT_GUIDE.md${NC}"
}

# 错误处理
trap 'print_error "启动过程中发生错误！"; exit 1' ERR

# 执行主函数
main "$@"
