# SimHub 设计文档索引 (Design Docs Index)

本文档库包含 SimHub 仿真资源中心的核心设计文档，旨在为开发人员、架构师及运维人员提供详尽的技术参考。

## 文档清单

| 编号 | 文档名称 | 描述 | 状态 |
| :--- | :--- | :--- | :--- |
| **01** | [架构概览 (Architecture Overview)](./01_architecture_overview.md) | 系统的整体愿景、设计理念、核心拓扑及技术选型。 | ✅ 已归档 |
| **02** | [资源管理模型 (Resource Management)](./02_resource_management.md) | 核心资源模型、生命周期事件、依赖管理及版本控制策略。 | ✅ 已归档 |
| **03** | [存储策略 (Storage Strategy)](./03_storage_strategy.md) | 基于 S3 (MinIO) 的存算分离设计、预签名上传流及目录规约。 | ✅ 已归档 |
| **04** | [全文检索与索引 (Search Engine)](./04_search_engine.md) | 基于 Elasticsearch 的全文搜索架构、Tika 文档提取及搜索高亮实现。 | ✅ 新增 |
| **05** | [认证与安全 (Authentication & Security)](./05_authentication_security.md) | 用户体系、RBAC 权限控制、令牌管理及安全拦截机制。 | ✅ 新增 |
| **06** | [前端工程体系 (Frontend Engineering)](./06_frontend_engineering.md) | Vue3 架构、组件化设计、可视化交互及性能优化策略。 | ✅ 已归档 |

## 编写原则 (Writing Principles)

遵循 "Design as Code" 理念，设计文档应与代码保持同步迭代。
- **清晰性 (Clarity)**: 优先使用图表 (Mermaid) 阐述复杂逻辑。
- **深度 (Depth)**: 不仅描述 "是什么"，更要解释 "为什么" (Design Rationale)。
- **一致性 (Consistency)**: 术语统一，中英对照明确。
