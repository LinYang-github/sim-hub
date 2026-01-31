# API 接口参考 (API Reference)

本文档定义了 SimHub Master API 的核心接口契约，供 SDK 及第三方集成开发参考。

## 1. 资源管理 (Resources)

### 查询资源列表
`GET /api/v1/resources`
- **参数**:
    - `type_key`: 资源类型过滤。
    - `keyword`: 关键词（名称、标签或正文）。
    - `category_id`: 分类过滤。
    - `scope`: `PUBLIC` 或 `PRIVATE`。
    - `page` / `size`: 分页参数。
- **返回**: 包含 `ResourceDTO` 数组及 `total` 总数。

### 获取资源详情
`GET /api/v1/resources/:id`
- **返回**: 资源的完整元数据及最新版本的下载链接。

## 2. 集成上传 (Integration Upload)

### 申请上传令牌
`POST /api/v1/integration/upload/token`
- **请求体**:
    ```json
    {
      "resource_type": "scenario",
      "filename": "demo.zip",
      "mode": "presigned"
    }
    ```
- **返回**: `ticket_id` 和 `presigned_url`。

### 确认上传完成
`POST /api/v1/integration/upload/confirm`
- **请求体**: 包含 `ticket_id`、`name`、`tags` 及 `semver`。
- **作用**: 触发后台处理流水线。

## 3. 分类与模型 (Categories & Types)

### 获取资源类型
`GET /api/v1/resource-types`
- **返回**: `modules.yaml` 中定义的类型及其 JSON Schema。

### 获取分类树
`GET /api/v1/categories?type_key=...`
- **返回**: 该类型下的多级目录结构。

## 4. 统计与仪表盘 (Stats)

### 获取概览数据
`GET /api/v1/dashboard/stats`
- **返回**: 各类型资源计数及最新更新列表。
