# SimHub - 仿真资源中心

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)
![Vue Version](https://img.shields.io/badge/vue-3.x-4FC08D.svg)
![MinIO](https://img.shields.io/badge/MinIO-Storage-C72C48.svg)

SimHub 是一个专为**仿真工程**设计的资源管理平台，提供高性能的仿真资源（想定、模型、地形）存储、版本控制、分类管理及自动化处理能力。它支持 **存算分离** 架构，可与 MinIO 对象存储和各类异构仿真处理器（Processor）无缝集成。

## 🌟 核心特性 (Key Features)

*   **📂 虚拟文件系统**: 支持多级资源分类（Tree/Flat 模式可配置），提供类似 Windows 文件管理器的操作体验。
*   **🏷️ 智能标签系统**: 支持自由打标，兼容 SQLite/MySQL，提供多维度资源检索能力。
*   **⚡️ 存算分离架构**:
    *   **Worker Pool**: 内置异步任务队列，削峰填谷，防止高并发上传导致服务崩溃。
    *   **STS 安全上传**: 支持 MinIO STS (Security Token Service) 临时凭证，前端直传存储桶，无需经由后端中转。
*   **🛡️ 数据高可靠性**:
    *   **Metadata Sidecar**: 核心元数据实时同步至 MinIO (`.meta.json`)，即使数据库丢失也能一键无损封禁。
    *   **自愈能力**: 提供 `SyncFromStorage` 接口，可随时从对象存储反向重建数据库索引。
*   **🔌 异构处理器集成**: 通过标准化 CLI 协议（JSON in/out）集成外部仿真工具（如 C++ 地形解析器、Python AI 模型分析器），自动提取资源元数据（文件数、时长、指纹等）。

## 🛠 技术栈 (Tech Stack)

*   **Backend**: Go (Gin, GORM, SQLite/MySQL), MinIO SDK
*   **Frontend**: Vue 3 (TypeScript, Element Plus, Vite)
*   **Storage**: MinIO (S3 Compatible)
*   **SDK**: C++ SDK (libcurl, nlohmann/json) for native integration

## 🚀 快速开始 (Getting Started)

### 环境依赖
*   Go 1.21+
*   Node.js 18+
*   MinIO Server (或使用 `minioadmin` 默认凭证的本地实例)

### 1. 启动后端 (Backend)

```bash
# 1. 确保 MinIO 已启动且凭证正确 (默认: minioadmin/minioadmin)
# 2. 运行服务 (自动迁移数据库结构 simhub.db)
go run cmd/simhub-api/main.go
```

服务默认运行在 `http://localhost:30030`。

### 2. 启动前端 (Frontend)

```bash
cd web
npm install
npm run dev
```

访问管理界面: `http://localhost:5173`

### 3. 运行 C++ SDK 示例 (可选)

SimHub 提供了标准 C++ SDK，用于仿真引擎集成：

```bash
cd sdk/cpp/examples/02_sts_upload
mkdir build && cd build
cmake ..
make
./sts_example
```

## ⚙️ 核心配置 (Configuration)

资源类型定义在 `config.yaml` 或数据库中管理。SimHub 启动时会根据配置自动注入基础类型：

```yaml
resource_types:
  - type_key: "scenario"
    type_name: "仿真想定"
    category_mode: "flat"      # 扁平模式，适合想定列表
    processor_cmd: "./drivers/scenario-processor" # 外部处理器路径
  - type_key: "model_glb"
    type_name: "3D模型"
    category_mode: "tree"      # 树形模式，适合模型库
```

## 📝 待办事项 (TODO)

- [x] **资源分类**: 实现多级虚拟文件夹目录树。
- [x] **标签系统**: 实现基于 SQLite JSON 的原子化标签管理。
- [x] **STS 上传**: 实现前端直传 MinIO，通过后端签发临时 Token。
- [x] **Worker Pool**: 实现异步资源处理任务队列。
- [x] **灾难恢复**: 实现 `SyncFromStorage` 和 Metadata Sidecar 机制。
- [x] **物理删除**: 实现数据库与 MinIO 文件的级联销毁。
- [ ] **MQ 集成**: 将本地 Processor 调用重构为消息队列模式 (Kafka/RabbitMQ)，实现真正的分布式处理。(Current: TODO logged in logs)
- [ ] **权限控制**: 集成 RBAC 角色权限管理。

## 📂 项目结构

```text
/
├── cmd/                # 应用程序入口 (API, CLI)
├── internal/
│   ├── conf/           # 配置定义
│   ├── data/           # 数据层 (GORM, MinIO Client)
│   ├── model/          # 领域模型 (Resource, Category, Version)
│   └── modules/        # 业务模块 (Resource Core Logic, Handlers)
├── pkg/
│   └── sts/            # MinIO STS 安全令牌服务封装
├── sdk/
│   └── cpp/            # C++ 客户端 SDK
└── web/                # Vue 3 前端工程
```

## 🤝 贡献

欢迎提交 Pull Request 或 Issue。对于重大变更，请先开启 Issue 讨论方案。

## 📄 许可证

MIT License
