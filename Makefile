.PHONY: build build-web build-terrain build-api clean dev

# 默认构建目标
build: build-web build-terrain build-demo build-api

# 构建主前端
build-web:
	@echo "Building main web..."
	cd web && npm install && npx vite build

# 构建地形管理前端
build-terrain:
	@echo "Building terrain web..."
	cd apps/terrain && npm install && npx vite build

# 构建演示模块
build-demo:
	@echo "Syncing demo-repo assets..."
	ls apps/demo-repo/index.html

# 构建后端 (会自动通过 go:embed 引用上述 dist 目录，需在 Go 代码中适配)
build-api:
	@echo "Collecting frontend assets..."
	rm -rf internal/ui/dist_web internal/ui/dist_terrain internal/ui/dist_demo
	mkdir -p internal/ui/dist_web internal/ui/dist_terrain internal/ui/dist_demo
	cp -r web/dist/* internal/ui/dist_web/
	cp -r apps/terrain/dist/* internal/ui/dist_terrain/
	cp -r apps/demo-repo/* internal/ui/dist_demo/
	@echo "Building Go API..."
	go build -o simhub-api ./cmd/simhub-api

# 清理构建产物
clean:
	rm -rf web/dist
	rm -rf apps/terrain/dist
	rm -f simhub-api

# 开发模式：启动所有服务 (建议分终端运行)
dev:
	@echo "Please run in separate terminals:"
	@echo "1. Backend: go run ./cmd/api"
	@echo "2. Main Web: cd web && npm run dev"
	@echo "3. Terrain Web: cd apps/terrain && npm run dev"
	@echo "4. Demo Web: ./run-demo-repo.sh"

# 运行演示模块
run-demo:
	./run-demo-repo.sh
