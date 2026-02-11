#!/bin/bash

# 部署脚本
# 使用方法: ./scripts/deploy.sh [环境] [版本]
# 示例: ./scripts/deploy.sh production latest

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 参数检查
ENVIRONMENT=${1:-production}
VERSION=${2:-latest}
PROJECT_NAME="go-progres"
BACKUP_DIR="/opt/backups/${PROJECT_NAME}"
LOG_DIR="/var/log/${PROJECT_NAME}"

log_info "开始部署 ${PROJECT_NAME} 到 ${ENVIRONMENT} 环境，版本: ${VERSION}"

# 检查必要的命令
check_dependencies() {
    log_info "检查依赖..."
    
    commands=("docker" "docker-compose" "curl" "git")
    for cmd in "${commands[@]}"; do
        if ! command -v $cmd &> /dev/null; then
            log_error "$cmd 未安装，请先安装"
            exit 1
        fi
    done
    
    log_success "依赖检查通过"
}

# 创建必要的目录
create_directories() {
    log_info "创建必要的目录..."
    
    sudo mkdir -p ${BACKUP_DIR}
    sudo mkdir -p ${LOG_DIR}
    sudo mkdir -p ./logs/nginx
    sudo mkdir -p ./monitoring/grafana/dashboards
    sudo mkdir -p ./monitoring/grafana/datasources
    sudo mkdir -p ./nginx/ssl
    sudo mkdir -p ./init-scripts
    
    # 设置权限
    sudo chown -R $USER:$USER ${BACKUP_DIR}
    sudo chown -R $USER:$USER ${LOG_DIR}
    
    log_success "目录创建完成"
}

# 备份当前版本
backup_current() {
    log_info "备份当前版本..."
    
    if docker-compose -f docker-compose.prod.yml ps -q | grep -q .; then
        BACKUP_NAME="${PROJECT_NAME}-$(date +%Y%m%d-%H%M%S)"
        sudo docker-compose -f docker-compose.prod.yml exec postgres pg_dump -U ${DATABASE_USER:-postgres} ${DATABASE_NAME:-postgres} > ${BACKUP_DIR}/${BACKUP_NAME}.sql
        log_success "数据库备份完成: ${BACKUP_DIR}/${BACKUP_NAME}.sql"
    else
        log_warning "没有运行中的服务，跳过数据库备份"
    fi
}

# 拉取最新代码
pull_code() {
    log_info "拉取最新代码..."
    
    git fetch origin
    git checkout main
    git pull origin main
    
    log_success "代码更新完成"
}

# 构建镜像
build_image() {
    log_info "构建 Docker 镜像..."
    
    if [ "$VERSION" = "latest" ]; then
        docker build -t ${PROJECT_NAME}:latest .
    else
        docker build -t ${PROJECT_NAME}:${VERSION} .
        docker tag ${PROJECT_NAME}:${VERSION} ${PROJECT_NAME}:latest
    fi
    
    log_success "镜像构建完成"
}

# 运行数据库迁移
run_migrations() {
    log_info "运行数据库迁移..."
    
    # 启动数据库服务
    docker-compose -f docker-compose.prod.yml up -d postgres redis
    
    # 等待数据库就绪
    log_info "等待数据库就绪..."
    until docker-compose -f docker-compose.prod.yml exec postgres pg_isready -U ${DATABASE_USER:-postgres}; do
        sleep 2
    done
    
    # 运行迁移
    docker-compose -f docker-compose.prod.yml run --rm app go run cmd/migrate/main.go
    
    log_success "数据库迁移完成"
}

# 部署服务
deploy_services() {
    log_info "部署服务..."
    
    # 设置环境变量
    export GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-"your-org/go-progres"}
    
    # 停止旧服务
    if docker-compose -f docker-compose.prod.yml ps -q | grep -q .; then
        log_info "停止旧服务..."
        docker-compose -f docker-compose.prod.yml down
    fi
    
    # 启动新服务
    docker-compose -f docker-compose.prod.yml up -d
    
    log_success "服务部署完成"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    # 等待服务启动
    sleep 30
    
    # 检查应用健康状态
    max_attempts=10
    attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            log_success "应用健康检查通过"
            break
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "健康检查失败，部署可能有问题"
            docker-compose -f docker-compose.prod.yml logs app
            exit 1
        fi
        
        log_warning "健康检查失败，重试中... ($attempt/$max_attempts)"
        sleep 10
        ((attempt++))
    done
    
    # 检查其他服务
    services=("postgres" "redis" "nginx")
    for service in "${services[@]}"; do
        if docker-compose -f docker-compose.prod.yml ps $service | grep -q "Up"; then
            log_success "$service 服务运行正常"
        else
            log_error "$service 服务运行异常"
            docker-compose -f docker-compose.prod.yml logs $service
        fi
    done
}

# 清理旧镜像
cleanup() {
    log_info "清理旧镜像..."
    
    # 删除未使用的镜像
    docker image prune -f
    
    # 删除超过7天的备份
    find ${BACKUP_DIR} -name "*.sql" -mtime +7 -delete 2>/dev/null || true
    
    log_success "清理完成"
}

# 发送通知
send_notification() {
    log_info "发送部署通知..."
    
    # 这里可以添加 Slack、钉钉、邮件等通知
    # 示例: curl -X POST -H 'Content-type: application/json' --data '{"text":"部署完成"}' YOUR_WEBHOOK_URL
    
    log_success "部署通知发送完成"
}

# 主函数
main() {
    log_info "开始部署流程..."
    
    check_dependencies
    create_directories
    backup_current
    pull_code
    build_image
    run_migrations
    deploy_services
    health_check
    cleanup
    send_notification
    
    log_success "🎉 部署完成！"
    log_info "应用访问地址: http://localhost:8080"
    log_info "监控面板: http://localhost:3000"
    log_info "Prometheus: http://localhost:9090"
}

# 错误处理
trap 'log_error "部署失败！"; docker-compose -f docker-compose.prod.yml logs app; exit 1' ERR

# 执行主函数
main "$@"
