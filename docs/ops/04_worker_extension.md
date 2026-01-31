# Worker 扩展开发说明 (Worker Extension)

SimHub 采用分布式异步处理架构。您可以根据业务需求，编写自定义 Worker 来处理特定类型的资源。

## 1. 核心原理
Worker 基于 **NATS 消息总线** 与 Master API 通信。Master 在资源发生变更时广播事件，Worker 消费事件并执行耗时处理工作，最后回传结果。

## 2. 扩展步骤

### 2.1 监听事件
订阅 NATS 主题：`simhub.resource.events.*`。
常见的事件消息包含：
- `resource_id`: 资源唯一标识。
- `version_id`: 触发变更的版本 ID。
- `type_key`: 资源类型（用于过滤本 Worker 是否感兴趣）。

### 2.2 获取原始数据
Worker 不直接连接数据库，而是通过 Master API 提供的预签名链接下载待处理的二进制文件。
- 调用 `GET /api/v1/resources/:id` 获取版本的 `download_url`。

### 2.3 执行处理业务
在本地执行具象业务，例如：
- **GIS 类型**: 解析 Shapefile 的空间参考系统 (SRS)。
- **图像类型**: 生成不同尺寸的预览缩略图。
- **文档类型**: 利用 OCR 或 Tika 提取全文内容。

### 2.4 回传处理结果
处理完成后，通过向 Master API 发送状态更新请求来告知结果：
- **请求方法**: `PATCH`
- **路径**: `/api/v1/resources/:id/process-result`
- **载荷 (JSON)**:
  ```json
  {
    "state": "ACTIVE",
    "meta_data": {
      "extracted_keywords": ["A", "B"],
      "page_count": 10
    },
    "message": "Processing success"
  }
  ```

## 3. 开发建议
- **幂等性**: 确保处理逻辑是幂等的，能够重试失败的任务。
- **超时控制**: 针对超大文件处理，应合理设置读取超时。
- **资源清理**: 处理完后及时清理 Worker 本地的临时解压目录。
