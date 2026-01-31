# 系统集成指南 (Integration Guide)

SimHub 作为一个开放平台，提供了多种方式供外部仿真引擎（如 Unreal Engine, Unity）或自动化脚本集成。

## 1. SDK 集成

### 1.1 C++ SDK
适用于仿真引擎运行时调用：
- **功能**: 自动处理预签名 URL 获取、异步下载、依赖包校验及加载通知。
- **协议**: 基于 RESTful 获取元数据，通过 S3 SDK 获取资源流。

### 1.2 Python SDK
适用于算法 Worker 或批量数据入库脚本。

## 2. API 集成规约

### 2.1 鉴权 (Authentication)
所有请求需在 Header 中包含：
`Authorization: Bearer <Your_Access_Token>`
令牌可在 Web 端的 **Token 设置** 页面生成。

### 2.2 核心 API 路径
- `POST /api/v1/integration/upload/token`: 申请上传凭证。
- `GET /api/v1/resources`: 检索资源列表。
- `GET /api/v1/resources/:id`: 获取资源详情及各版本链接。

## 3. 事件总线集成 (NATS)

外部系统可以订阅 NATS 消息来实时感知资源的生命周期变化：
- 订阅主题: `simhub.resource.events.*`
- 事件载荷 (JSON):
```json
{
  "event_type": "version.activated",
  "resource_id": "...",
  "version_id": "...",
  "timestamp": "..."
}
```

## 4. 存储层集成
虽然建议通过 API 操作，但在特定高性能场景下，仿真节点可以直接挂载 S3 存储进行 **Zero-Copy** 加载。
