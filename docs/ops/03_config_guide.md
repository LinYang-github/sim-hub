# 配置文件详细说明 (Configuration Guide)

SimHub 采用声明式配置，通过 YAML 文件定义系统基础架构连接及业务逻辑模型。

## 1. 系统配置文件 (`config-api.yaml` / `config-worker.yaml`)

这两个文件定义了基础设施的连接细节及服务的基础运行参数。

### 1.1 核心字段说明 (Infrastructure)
| 字段 | 描述 | 示例值 |
| :--- | :--- | :--- |
| **`server.port`** | API 服务监听端口 | `30030` |
| **`server.api_base_url`** | 服务的外部访问地址，用于生成下载链接等 | `http://localhost:30030` |
| **`data.db.dsn`** | 数据库连接串 (DSN) | `simhub.db?_pragma=foreign_keys(1)` |
| **`data.storage.endpoint`** | MinIO/S3 访问地址 | `localhost:9000` |
| **`data.storage.bucket`** | 默认存储桶名称 | `simhub-raw` |
| **`data.nats.url`** | NATS 消息总线地址 | `nats://localhost:4222` |
| **`data.elasticsearch.url`** | Elasticsearch 服务地址 | `http://localhost:9200` |
| **`data.tika.url`** | Apache Tika 服务地址 (用于文本提取) | `http://localhost:9998` |

## 2. 业务模型配置 (`modules.yaml`)

`modules.yaml` 定义了系统支持的资源类型及其展现方式。这是 SimHub 灵活性的核心。

### 2.1 资源类型属性
- **`type_key`**: 资源的唯一主键标识（如 `model`, `scenario`, `documents`）。
- **`type_name`**: 在前端 UI 展示的友好名称。
- **`category_mode`**: 
    - `TREE`: 启用类似文件系统的多级分类树。
    - `FLAT`: 仅支持一级扁平分类标签。
- **`schema_def`**: 
    - 采用 **JSON Schema** 格式。
    - 定义了该类型资源特有的元数据字段（如：仿真想定所需的“步长”、“地理范围”）。
    - 前端会自动根据此定义渲染出表单编辑器。

### 2.2 处理与预览配置
- **`upload_mode`**: 
    - `FILE`: 单文件上传。
    - `FOLDER`: 文件夹上传（前端自动打包为 ZIP）。
- **`process_conf.viewer`**: 指定该类型资源首选的前端预览组件（如 `CesiumPreview`, `PDFPreview`）。
- **`process_conf.pipeline`**: 指定上传完成后触发的后端处理流水线名称。
