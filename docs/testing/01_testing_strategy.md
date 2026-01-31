# 测试体系概览 (Testing Strategy)

SimHub 采用分层测试策略，确保从核心逻辑到外部接口的稳健性。

## 1. 测试层级

### 1.1 单元测试 (Unit Tests)
- **范围**: 位于各模块内部，如 `internal/modules/resource/core`。
- **重点**: 验证资源管理状态机、版本控制逻辑、依赖树解析及元数据合成。
- **技术**: 使用 `testing` 包，配合 `testify/assert` 和 `mock` 模拟外部依赖（如存储接口、NATS 客户端）。

### 1.2 集成测试 (Integration Tests)
- **范围**: 验证 API 接口与数据库、认证中间件的配合。
- **重点**: 模拟真实的 Gin 路由链路，执行带 Auth 头的请求，验证 CRUD 的完整性及权限拦截（RBAC）。
- **示例**: `internal/modules/resource/api_test.go`。

### 1.3 端到端测试 (E2E Tests)
- **范围**: 位于 `tests/e2e`。
- **重点**: 验证全链路协同，包括文件上传到 MinIO、Worker 异步处理、ES 索引建立及最终的前端搜索呈现。

## 2. 验证规约

### 2.1 执行全量测试
```bash
go test ./...
```

### 2.2 性能/压力测试 (Benchmarking)
- **路径**: `tests/stress`。
- **重点**: 验证百 GB 级超大文件上传的内存稳定性及高并发搜索下的 ES 响应延迟。

## 3. 持续集成 (CI)
建议在 CI 流程中配置以下步骤：
- 代码格式校验 (go fmt)
- 静态分析 (go vet, staticcheck)
- 单元测试与覆盖率报告
