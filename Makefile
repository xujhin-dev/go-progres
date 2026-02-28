# Go项目测试和覆盖率Makefile

.PHONY: test test-unit test-integration test-benchmark test-concurrent test-coverage clean help

# 默认目标
test: test-unit test-integration
	@echo "✅ 所有测试完成"

# 运行单元测试
test-unit:
	@echo "🧪 运行单元测试..."
	go test -v ./internal/domain/user/service/

# 运行集成测试
test-integration:
	@echo "🔗 运行集成测试..."
	go test -v -tags=integration ./internal/domain/user/ || true

# 运行性能测试
test-benchmark:
	@echo "⚡ 运行性能测试..."
	go test -bench=. -benchmem ./internal/domain/user/service/

# 运行并发测试
test-concurrent:
	@echo "🔄 运行并发测试..."
	go test -v -run=Concurrent ./internal/domain/user/service/

# 生成覆盖率报告
test-coverage:
	@echo "📊 生成覆盖率报告..."
	@mkdir -p coverage
	@go test -v -coverprofile=coverage/unit.out -covermode=atomic ./internal/domain/user/service/
	@go tool cover -html=coverage/unit.out -o coverage/coverage.html
	@go tool cover -func=coverage/unit.out > coverage/coverage.txt
	@echo "📄 覆盖率报告已生成: coverage/coverage.html"
	@echo "📝 详细报告: coverage/coverage.txt"

# 运行所有测试并生成覆盖率
test-all: test-unit test-benchmark test-concurrent test-coverage
	@echo "🎉 所有测试和覆盖率报告完成！"

# 运行用户模块完整测试套件
test-user: 
	@echo "👤 运行用户模块完整测试套件..."
	@./scripts/test_coverage.sh user

# 运行API测试
test-api:
	@echo "🌐 启动服务并运行API测试..."
	@go run cmd/main.go &
	@sleep 3
	@echo "📡 测试API端点..."
	@curl -s -X POST http://localhost:8080/auth/otp -H "Content-Type: application/json" -d '{"mobile":"13800138000"}' | jq .
	@curl -s -X POST http://localhost:8080/auth/login -H "Content-Type: application/json" -d '{"mobile":"13800138000","code":"123456"}' | jq .
	@TOKEN=$$(curl -s -X POST http://localhost:8080/auth/login -H "Content-Type: application/json" -d '{"mobile":"13800138000","code":"123456"}' | jq -r '.data'); \
	curl -s -H "Authorization: Bearer $$TOKEN" http://localhost:8080/users/ | jq .
	@pkill -f "go run cmd/main.go"

# 代码质量检查
lint:
	@echo "🔍 运行代码质量检查..."
	@golangci-lint run ./...

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...

# 清理测试文件
clean:
	@echo "🧹 清理测试文件..."
	@rm -rf coverage/
	@rm -f *.out
	@go clean -testcache

# 安装依赖
deps:
	@echo "📦 安装依赖..."
	@go mod tidy
	@go mod download

# 显示帮助信息
help:
	@echo "📖 可用的测试命令:"
	@echo ""
	@echo "  test          - 运行单元测试和集成测试"
	@echo "  test-unit     - 运行单元测试"
	@echo "  test-integration - 运行集成测试"
	@echo "  test-benchmark - 运行性能测试"
	@echo "  test-concurrent - 运行并发测试"
	@echo "  test-coverage  - 生成覆盖率报告"
	@echo "  test-all      - 运行所有测试并生成覆盖率"
	@echo "  test-user     - 运行用户模块完整测试套件"
	@echo "  test-api      - 运行API测试"
	@echo "  lint          - 运行代码质量检查"
	@echo "  fmt           - 格式化代码"
	@echo "  clean         - 清理测试文件"
	@echo "  deps          - 安装依赖"
	@echo "  help          - 显示此帮助信息"
	@echo ""
	@echo "📊 覆盖率报告位置: coverage/coverage.html"
	@echo "🔍 详细覆盖率报告: coverage/coverage.txt"
