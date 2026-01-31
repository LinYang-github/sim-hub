# 03 存储策略 (Storage Strategy)

## 1. 存算分离架构
SimHub 采用 S3 协议作为存储标准，后端不直接接触大规模二进制流。

## 2. 上传工作流 (The Upload Flow)

### 2.1 预签名 URL 流程 (Presigned URL)
对于大多数中等规模资源，采用预签名 PUT 上传：
1. **Request**: 前端向 Master 请求上传令牌（文件名、MD5）。
2. **Issue**: Master 计算存储路径并向 MinIO 获取预签名 URL。
3. **Upload**: 前端直接使用 `axios.put` 将数据流推送到 MinIO。**注意：此步骤需剔除自定义的 Authorization 头以免签名冲突。**
4. **Confirm**: 上传完成后，前端请求 Master 确认，Master 触发后续 Worker 链路。

### 2.2 STS 临时凭证流程
对于 GB 级大型文件夹或复杂分片上传，支持发放 STS 临时权限：
1. Master 为用户生成 30min 内有效的临时 AK/SK。
2. 前端集成 S3 SDK 直接操作分片上传流程。

## 3. 存储路径规约
遵循语义化和可发现原则：
```text
simhub-raw/ (存储桶名称)
  ├── {type_key}/
  │   └── {resource_id}/
  │       ├── v{version_num}/
  │       │   ├── {filename}
  │       │   └── .meta.json (Sidecar 文件)
```

- **Sidecar 设计**: 每一份资源旁附带其元数据备份，确保存储层具备“自描述”能力。当数据库索引丢失时，可通过扫描存储桶快速重建。

## 4. 文件名处理与安全性
- **编码 (Encoding)**: 所有原始文件名在存储路径中均通过 `encodeURIComponent` 处理，解决中文及特殊字符在 S3 signature 过程中的冲突。
- **一致性**: 路径中包含版本前缀，确保物理文件的不可变性。
