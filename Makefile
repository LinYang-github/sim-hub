.PHONY: build build-web build-terrain build-api clean dev

# 默认构建目标
build: build-web build-ext-apps build-api

# 构建主前端
build-web:
	@echo "Building main web..."
	cd web && npm install && npx vite build


# 构建所有扩展插件示例 (Demos)
build-ext-apps:
	@echo "Building consolidated external apps..."
	cd apps/ext-apps && npm install && npx vite build

# 构建后端 (会自动通过 go:embed 引用上述 dist 目录)
build-api:
	@echo "Collecting frontend assets..."
	rm -rf internal/ui/dist_web internal/ui/dist_ext_apps
	mkdir -p internal/ui/dist_web internal/ui/dist_ext_apps
	cp -r web/dist/* internal/ui/dist_web/
	cp -r apps/ext-apps/dist/* internal/ui/dist_ext_apps/
	@echo "Building Go API..."
	go build -o simhub-api ./cmd/simhub-api

# 清理构建产物
clean:
	rm -rf web/dist
	rm -f simhub-api

# 开发模式：启动所有服务 (建议分终端运行)
dev:
	@echo "Please run in separate terminals:"
	@echo "1. Backend: go run ./cmd/simhub-api/main.go"
	@echo "2. Main Web: cd web && npm run dev"
	@echo "3. Example Apps: cd apps/ext-apps && npm run dev"

# 运行所有扩展应用 (单端口模式)
run-ext:
	cd apps/ext-apps && npm run dev

# 自动化测试相关
.PHONY: test test-backend test-frontend

test: test-backend test-frontend

test-backend:
	@echo "Running backend unit and integration tests..."
	go test -v ./internal/...

test-frontend:
	@echo "Running frontend vitest..."
	cd web && npm run test -- --run
