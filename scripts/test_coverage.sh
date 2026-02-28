#!/bin/bash

# 测试覆盖率报告脚本
# 用法: ./scripts/test_coverage.sh [module]

set -e

MODULE=${1:-"user"}

echo "🧪 运行测试并生成覆盖率报告..."
echo "📦 模块: $MODULE"

# 创建覆盖率目录
mkdir -p coverage

# 运行单元测试并生成覆盖率
echo "🔬 运行单元测试..."
go test -v -coverprofile=coverage/unit.out -covermode=atomic ./internal/domain/$MODULE/service/

# 运行性能测试
echo "⚡ 运行性能测试..."
go test -bench=. -benchmem ./internal/domain/$MODULE/service/ > coverage/benchmark.txt 2>&1

# 运行并发测试
echo "🔄 运行并发测试..."
go test -v -run=Concurrent ./internal/domain/$MODULE/service/ > coverage/concurrent.txt 2>&1

# 生成HTML覆盖率报告
echo "📊 生成HTML覆盖率报告..."
go tool cover -html=coverage/unit.out -o coverage/coverage.html

# 生成覆盖率统计
echo "📈 生成覆盖率统计..."
go tool cover -func=coverage/unit.out > coverage/coverage.txt

# 显示覆盖率摘要
echo ""
echo "🎯 覆盖率报告摘要:"
echo "=================================="

if [ -f "coverage/coverage.txt" ]; then
    echo "📝 详细覆盖率报告: coverage/coverage.txt"
    
    # 计算总覆盖率
    TOTAL_COVERAGE=$(go tool cover -func=coverage/unit.out | grep "total:" | awk '{print $3}' | sed 's/%//')
    if [ ! -z "$TOTAL_COVERAGE" ]; then
        echo "📊 总覆盖率: ${TOTAL_COVERAGE}%"
    fi
    
    # 显示低覆盖率函数
    echo ""
    echo "⚠️  低覆盖率函数 (<80%):"
    go tool cover -func=coverage/unit.out | awk -F'\t' 'NR>1 && $3 != "" && ($3+0) < 80 {print "  • " $1 ": " $3 "%"}'
fi

echo ""
echo "📄 生成的文件:"
echo "  • coverage/coverage.html - HTML覆盖率报告"
echo "  • coverage/coverage.txt - 文本覆盖率报告"
echo "  • coverage/benchmark.txt - 性能测试结果"
echo "  • coverage/concurrent.txt - 并发测试结果"

echo ""
echo "🌐 在浏览器中打开HTML报告:"
echo "open coverage/coverage.html"

# 检查覆盖率阈值
if [ ! -z "$TOTAL_COVERAGE" ]; then
    if (( $(echo "$TOTAL_COVERAGE >= 80" | bc -l) )); then
        echo "✅ 覆盖率达标 (≥80%)"
    else
        echo "❌ 覆盖率未达标 (<80%)"
        exit 1
    fi
fi

echo ""
echo "🎉 测试完成！"
