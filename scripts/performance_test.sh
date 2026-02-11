#!/bin/bash

# 性能测试脚本
# 使用方法: ./scripts/performance_test.sh [测试类型]

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

# 默认配置
BASE_URL="http://localhost:8080"
TEST_TYPE="${1:-all}"
CONCURRENCY=50
DURATION=30

# 显示帮助
show_help() {
    echo "Go Progress 性能测试脚本"
    echo "================================"
    echo "用法: $0 [测试类型] [选项]"
    echo ""
    echo "测试类型:"
    echo "  health     - 健康检查测试"
    echo "  api        - API 性能测试"
    echo "  load       - 负载测试"
    echo "  stress     - 压力测试"
    echo "  benchmark  - 基准测试"
    echo "  all        - 运行所有测试 (默认)"
    echo ""
    echo "环境变量:"
    echo "  BASE_URL     - 测试服务器地址 (默认: http://localhost:8080)"
    echo "  CONCURRENCY  - 并发数 (默认: 50)"
    echo "  DURATION     - 测试时长秒 (默认: 30)"
    echo ""
    echo "示例:"
    echo "  $0 health"
    echo "  $0 api"
    echo "  BASE_URL=http://localhost:8080 CONCURRENCY=100 $0 load"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装"
        exit 1
    fi
    
    if ! command -v wrk &> /dev/null; then
        log_warning "wrk 未安装，将使用 curl 进行简单测试"
    fi
    
    log_success "依赖检查完成"
}

# 检查服务器状态
check_server() {
    log_info "检查服务器状态: $BASE_URL"
    
    if ! curl -s --max-time 5 "$BASE_URL/health" > /dev/null; then
        log_error "服务器不可用: $BASE_URL"
        log_info "请确保服务器正在运行: ./bin/server"
        exit 1
    fi
    
    log_success "服务器可用"
}

# 健康检查测试
test_health() {
    log_info "运行健康检查性能测试..."
    
    echo "📊 健康检查测试结果"
    echo "================================"
    
    # 使用 curl 进行简单测试
    start_time=$(date +%s%N)
    success_count=0
    total_requests=100
    
    for ((i=1; i<=total_requests; i++)); do
        if curl -s --max-time 2 "$BASE_URL/health" > /dev/null; then
            ((success_count++))
        fi
    done
    
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000)) # 转换为毫秒
    
    qps=$(echo "scale=2; $success_count * 1000 / $duration" | bc -l)
    success_rate=$(echo "scale=2; $success_count * 100 / $total_requests" | bc -l)
    avg_time=$(echo "scale=2; $duration / $success_count" | bc -l)
    
    echo "总请求数: $total_requests"
    echo "成功请求: $success_count"
    echo "失败请求: $((total_requests - success_count))"
    echo "QPS: $qps"
    echo "成功率: $success_rate%"
    echo "平均响应时间: ${avg_time}ms"
    echo "================================"
}

# API 性能测试
test_api() {
    log_info "运行 API 性能测试..."
    
    echo "📊 API 性能测试结果"
    echo "================================"
    
    # 测试健康检查
    test_endpoint "/health" "健康检查"
    
    # 测试用户列表 (预期 401)
    test_endpoint "/users/" "用户列表"
    
    # 测试登录 (预期 400)
    test_endpoint_login "/auth/login" "登录接口"
    
    echo "================================"
}

# 测试单个端点
test_endpoint() {
    local endpoint="$1"
    local name="$2"
    
    echo "测试端点: $name ($endpoint)"
    
    start_time=$(date +%s%N)
    success_count=0
    total_requests=50
    
    for ((i=1; i<=total_requests; i++)); do
        if curl -s --max-time 2 "$BASE_URL$endpoint" > /dev/null; then
            ((success_count++))
        fi
    done
    
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    
    qps=$(echo "scale=2; $success_count * 1000 / $duration" | bc -l 2>/dev/null || echo "0")
    success_rate=$(echo "scale=2; $success_count * 100 / $total_requests" | bc -l 2>/dev/null || echo "0")
    
    echo "  QPS: $qps, 成功率: $success_rate%, 总耗时: ${duration}ms"
}

# 测试登录端点
test_endpoint_login() {
    local endpoint="$1"
    local name="$2"
    
    echo "测试端点: $name ($endpoint)"
    
    start_time=$(date +%s%N)
    success_count=0
    total_requests=30
    
    for ((i=1; i<=total_requests; i++)); do
        if curl -s --max-time 2 \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{"mobile":"13800138000","code":"123456"}' \
            "$BASE_URL$endpoint" > /dev/null; then
            ((success_count++))
        fi
    done
    
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    
    qps=$(echo "scale=2; $success_count * 1000 / $duration" | bc -l 2>/dev/null || echo "0")
    success_rate=$(echo "scale=2; $success_count * 100 / $total_requests" | bc -l 2>/dev/null || echo "0")
    
    echo "  QPS: $qps, 成功率: $success_rate%, 总耗时: ${duration}ms"
}

# 负载测试
test_load() {
    log_info "运行负载测试..."
    
    if command -v wrk &> /dev/null; then
        echo "📊 使用 wrk 进行负载测试"
        echo "================================"
        
        # 健康检查负载测试
        echo "健康检查负载测试 (10并发, 30秒):"
        wrk -t4 -c10 -d30s --timeout 10s "$BASE_URL/health"
        
        echo ""
        echo "用户列表负载测试 (5并发, 30秒):"
        wrk -t2 -c5 -d30s --timeout 10s "$BASE_URL/users/"
    else
        log_warning "wrk 未安装，使用 curl 进行简单负载测试"
        test_health
    fi
}

# 压力测试
test_stress() {
    log_info "运行压力测试..."
    
    echo "📊 压力测试结果"
    echo "================================"
    
    # 渐进式压力测试
    for concurrency in 10 20 50 100; do
        echo "测试并发数: $concurrency"
        
        start_time=$(date +%s%N)
        success_count=0
        total_requests=$((concurrency * 10))
        
        # 并发执行
        for ((i=1; i<=concurrency; i++)); do
            (
                local local_success=0
                for ((j=1; j<=10; j++)); do
                    if curl -s --max-time 3 "$BASE_URL/health" > /dev/null; then
                        ((local_success++))
                    fi
                done
                echo $local_success
            ) &
        done
        
        # 等待所有后台任务完成
        wait
        
        end_time=$(date +%s%N)
        duration=$((($end_time - $start_time) / 1000000))
        
        echo "  并发数: $concurrency, 总耗时: ${duration}ms"
        
        # 如果错误率过高，停止测试
        if [ $duration -gt $((concurrency * 100)) ]; then
            echo "  ⚠️ 检测到性能瓶颈，停止压力测试"
            break
        fi
    done
    
    echo "================================"
}

# 基准测试
test_benchmark() {
    log_info "运行基准测试..."
    
    echo "📊 基准测试结果"
    echo "================================"
    
    # 测试单次请求延迟
    echo "单次请求延迟测试 (100次):"
    
    total_time=0
    for ((i=1; i<=100; i++)); do
        start=$(date +%s%N)
        curl -s --max-time 2 "$BASE_URL/health" > /dev/null
        end=$(date +%s%N)
        time=$((($end - $start) / 1000000)) # 转换为毫秒
        total_time=$((total_time + time))
    done
    
    avg_time=$((total_time / 100))
    echo "  平均延迟: ${avg_time}ms"
    
    # 测试连接建立时间
    echo "连接建立时间测试 (50次):"
    
    total_connect_time=0
    for ((i=1; i<=50; i++)); do
        output=$(curl -w "%{time_connect}" -s --max-time 2 "$BASE_URL/health" -o /dev/null)
        total_connect_time=$(echo "$total_connect_time + $output" | bc -l)
    done
    
    avg_connect_time=$(echo "scale=3; $total_connect_time / 50" | bc -l)
    echo "  平均连接时间: ${avg_connect_time}s"
    
    echo "================================"
}

# 响应时间测试
test_response() {
    log_info "运行响应时间分布测试..."
    
    echo "📊 响应时间分布测试"
    echo "================================"
    
    # 收集响应时间数据
    response_times=()
    total_requests=200
    
    for ((i=1; i<=total_requests; i++)); do
        start=$(date +%s%N)
        if curl -s --max-time 2 "$BASE_URL/health" > /dev/null; then
            end=$(date +%s%N)
            time=$((($end - $start) / 1000000)) # 转换为毫秒
            response_times+=($time)
        fi
    done
    
    # 计算统计信息
    if [ ${#response_times[@]} -gt 0 ]; then
        # 排序
        IFS=$'\n' sorted=($(sort -n <<<"${response_times[*]}"))
        unset IFS
        
        count=${#sorted[@]}
        
        # 计算百分位数
        min=${sorted[0]}
        max=${sorted[$((count-1))]}
        
        p50_index=$((count * 50 / 100))
        p95_index=$((count * 95 / 100))
        p99_index=$((count * 99 / 100))
        
        p50=${sorted[$p50_index]}
        p95=${sorted[$p95_index]}
        p99=${sorted[$p99_index]}
        
        # 计算平均值
        total=0
        for time in "${sorted[@]}"; do
            total=$((total + time))
        done
        avg=$((total / count))
        
        echo "样本数量: $count"
        echo "最小延迟: ${min}ms"
        echo "最大延迟: ${max}ms"
        echo "平均延迟: ${avg}ms"
        echo "P50: ${p50}ms"
        echo "P95: ${p95}ms"
        echo "P99: ${p99}ms"
    else
        echo "没有成功的请求"
    fi
    
    echo "================================"
}

# 运行所有测试
run_all_tests() {
    log_info "运行完整性能测试套件"
    echo "================================"
    
    echo "🎯 Go Progress 性能测试报告"
    echo "测试时间: $(date)"
    echo "测试服务器: $BASE_URL"
    echo "================================"
    
    test_health
    echo ""
    
    test_api
    echo ""
    
    test_benchmark
    echo ""
    
    test_response
    echo ""
    
    test_load
    echo ""
    
    test_stress
    echo ""
    
    echo "🎉 完整性能测试完成！"
    echo "================================"
    echo "📝 性能评估建议:"
    echo "1. P95 响应时间应 < 100ms"
    echo "2. 错误率应 < 0.1%"
    echo "3. QPS 应根据业务需求评估"
    echo "4. 监控系统资源使用情况"
    echo "5. 定期进行性能回归测试"
}

# 主函数
main() {
    echo "🚀 Go Progress 性能测试工具"
    echo "================================"
    
    # 检查参数
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        show_help
        exit 0
    fi
    
    # 检查依赖
    check_dependencies
    
    # 检查服务器
    check_server
    
    # 根据测试类型运行相应的测试
    case "$TEST_TYPE" in
        "health")
            test_health
            ;;
        "api")
            test_api
            ;;
        "load")
            test_load
            ;;
        "stress")
            test_stress
            ;;
        "benchmark")
            test_benchmark
            ;;
        "response")
            test_response
            ;;
        "all")
            run_all_tests
            ;;
        *)
            log_error "未知的测试类型: $TEST_TYPE"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
