# 常见问题与排障 (Troubleshooting & FAQ)

本指南汇总了在开发、部署及利用 SimHub 进行仿真集成时的常见问题及其解决方案。

## 1. 存储与上传 (Storage & Upload)

### Q: 上传文件到 MinIO 时返回 `400 Bad Request`
- **原因**: 预签名 URL (Presigned URL) 包含签名信息。如果请求头中携带了全局注入的 `Authorization` (JWT)，S3 协议会认为身份冲突而拒绝。
- **解决**: 在上传到 S3 存储桶的逻辑中，必须创建一个不带全局拦截器的临时 `axios` 实例，确保请求头简洁。

### Q: 上传包含特殊字符（如 `#`, `&`, `+`）的文件失败
- **原因**: S3 路径签名对字符敏感。
- **解决**: 所有的物理路径（特别是文件名部分）在发送 PUT 请求前，必须经过 `encodeURIComponent` 处理。

## 2. 搜索与索引 (Search & Index)

### Q: 搜索资源名的前几个字母搜不到结果
- **原因**: Elasticsearch (ES) 默认的分词器（如 standard 或 ik）通常以词为单位。搜索 "tri" 可能匹配不到 "triangle"。
- **解决**: SimHub 采用了 **Hybrid Search** 策略。系统会同时发起 ES 全文检索和 SQL `LIKE` 模糊匹配，并取并集。

### Q: 上传了 PDF/Doc 文档，但搜不到其中的正文内容
- **原因**: 索引具有延迟，或者 Worker 未正常启动。
- **排查**:
    1. 检查 `simhub-worker` 是否处于运行状态并已连接到 NATS。
    2. 查看 Worker 日志是否提示 `Tika connection failed`（确保 Tika 容器已启动）。
    3. 确保 `modules.yaml` 中该资源类型配置了正确的 `pipeline: "es_index"`。

## 3. 消息与事件 (NATS)

### Q: 订阅了事件但没有收到通知
- **排查**:
    1. 确认 NATS Server 地址配置正确（默认 `4222`）。
    2. Master API 启动日志中应显示 `Connected to NATS`。
    3. 只有当资源状态变为 `ACTIVE` (处理完成) 时，系统才会通过 `simhub.resource.events.activated` 发送通知。

## 4. 前端样式 (Frontend)

### Q: 修改了 CSS 导致白屏或编译报错 `unmatched "}"`
- **原因**: SCSS 嵌套层级错误。
- **排查**: 检查 `src/components/common/GlobalSearch.vue` 等大型 Vue 组件的 `<style>` 块，确保嵌套级别完整。
