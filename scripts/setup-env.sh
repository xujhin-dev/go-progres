#!/bin/bash

# 环境配置设置脚本
# 使用方法: ./scripts/setup-env.sh [环境]
# 示例: ./scripts/setup-env.sh production

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

ENVIRONMENT=${1:-development}
PROJECT_NAME="go-progres"

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

# 生成随机密码
generate_password() {
    openssl rand -base64 32 | tr -d "=+/" | cut -c1-25
}

# 生成 JWT Secret
generate_jwt_secret() {
    openssl rand -base64 64 | tr -d "=+/" | cut -c1-64
}

# 设置开发环境
setup_development() {
    log_info "设置开发环境配置..."
    
    cat > .env << EOF
# 开发环境配置
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=postgres

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

JWT_SECRET=$(generate_jwt_secret)
JWT_EXPIRE=24

GIN_MODE=debug
SERVER_PORT=8080

# 监控配置
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin
EOF

    log_success "开发环境配置完成"
}

# 设置测试环境
setup_testing() {
    log_info "设置测试环境配置..."
    
    cat > .env << EOF
# 测试环境配置
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres_test
DATABASE_PASSWORD=$(generate_password)
DATABASE_NAME=postgres_test

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=$(generate_password)

JWT_SECRET=$(generate_jwt_secret)
JWT_EXPIRE=1

GIN_MODE=test
SERVER_PORT=8080

# 监控配置
GRAFANA_USER=admin
GRAFANA_PASSWORD=$(generate_password)
EOF

    log_success "测试环境配置完成"
}

# 设置生产环境
setup_production() {
    log_info "设置生产环境配置..."
    
    if [ ! -f .env ]; then
        log_warning "未找到 .env 文件，基于模板创建..."
        cp .env.example .env
    fi
    
    # 生成安全的生产环境密码
    DB_PASSWORD=$(generate_password)
    REDIS_PASSWORD=$(generate_password)
    JWT_SECRET=$(generate_jwt_secret)
    GRAFANA_PASSWORD=$(generate_password)
    
    log_info "请手动更新 .env 文件中的以下配置："
    echo ""
    echo -e "${YELLOW}重要安全配置:${NC}"
    echo "DATABASE_PASSWORD=${DB_PASSWORD}"
    echo "REDIS_PASSWORD=${REDIS_PASSWORD}"
    echo "JWT_SECRET=${JWT_SECRET}"
    echo "GRAFANA_PASSWORD=${GRAFANA_PASSWORD}"
    echo ""
    echo -e "${YELLOW}其他需要配置的项目:${NC}"
    echo "- GITHUB_REPOSITORY (你的 GitHub 仓库)"
    echo "- OSS 配置 (如果需要文件上传)"
    echo "- 支付配置 (如果需要支付功能)"
    echo "- 通知配置 (如果需要通知功能)"
    echo ""
    
    log_info "请将这些配置更新到 .env 文件中"
}

# 验证配置
validate_config() {
    log_info "验证配置..."
    
    if [ ! -f .env ]; then
        log_error ".env 文件不存在"
        return 1
    fi
    
    # 检查必需的环境变量
    required_vars=("DATABASE_HOST" "DATABASE_USER" "DATABASE_PASSWORD" "DATABASE_NAME" "JWT_SECRET")
    
    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" .env; then
            log_error "缺少必需的环境变量: $var"
            return 1
        fi
    done
    
    log_success "配置验证通过"
}

# 创建 SSL 证书 (开发用)
create_ssl_cert() {
    log_info "创建开发用 SSL 证书..."
    
    mkdir -p nginx/ssl
    
    if [ ! -f nginx/ssl/cert.pem ]; then
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout nginx/ssl/key.pem \
            -out nginx/ssl/cert.pem \
            -subj "/C=CN/ST=State/L=City/O=Organization/CN=localhost"
        
        log_success "SSL 证书创建完成"
    else
        log_info "SSL 证书已存在，跳过创建"
    fi
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    
    # 创建数据库初始化脚本
    mkdir -p init-scripts
    
    cat > init-scripts/01-init.sql << EOF
-- 创建数据库用户和权限
CREATE USER IF NOT EXISTS ${PROJECT_NAME} WITH PASSWORD '${DATABASE_PASSWORD:-postgres}';
GRANT ALL PRIVILEGES ON DATABASE ${DATABASE_NAME:-postgres} TO ${PROJECT_NAME};

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
EOF

    log_success "数据库初始化脚本创建完成"
}

# 主函数
main() {
    log_info "开始设置 ${ENVIRONMENT} 环境..."
    
    case $ENVIRONMENT in
        "development"|"dev")
            setup_development
            ;;
        "testing"|"test")
            setup_testing
            ;;
        "production"|"prod")
            setup_production
            ;;
        *)
            log_error "不支持的环境: $ENVIRONMENT"
            echo "支持的环境: development, testing, production"
            exit 1
            ;;
    esac
    
    validate_config
    create_ssl_cert
    init_database
    
    log_success "🎉 环境配置完成！"
    log_info "配置文件: .env"
    log_info "请检查并根据需要修改配置"
}

# 执行主函数
main "$@"
