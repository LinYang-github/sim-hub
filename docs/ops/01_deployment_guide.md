# 部署与运维手册 (Operations Guide)

## 1. 快速部署 (Quick Start)

SimHub 深度依赖 Docker 生态，推荐使用 `docker-compose` 快速搭建基础设施。

### 1.1 环境要求
- Docker 20.10+
- Docker Compose v2.0+
- 推荐配置：4 Core CPU / 8GB RAM / 100GB+ SSD

### 1.2 启动基础设施
```bash
docker-compose up -d
```
该命令将启动：
- **MinIO**: 对象存储 (9000端口)
- **NATS**: 消息总线 (4222端口)
- **Elasticsearch**: 搜索引擎 (9200端口)
- **Apache Tika**: 文档提取引擎 (9998端口)

## 2. 环境配置 (Configuration)

系统通过 `config-api.yaml` 和 `config-worker.yaml` 进行精细化配置。

### 核心配置项说明
- `server.port`: API 服务端口（默认 30030）。
- `data.db.dsn`: 数据库连接串（支持 SQLite/MySQL）。
- `data.storage`: MinIO 端点、密钥及存储桶名称。
- `data.elasticsearch`: ES 连接地址。
- `worker.process_conf`: 定义 Worker 的并发处理能力。

## 3. 日志与监控

### 3.1 日志查看
- API 日志: `tail -f api.log`
- Worker 日志: `tail -f worker.log`
- 基础设施日志: `docker-compose logs -f`

### 3.2 健康检查
- API 健康检查: `GET /health` (待实现) 或观察控制台 GIN 启动输出。
- 基础设施健康: 访问 MinIO 控制台确认存储桶状态。

## 4. 备份与恢复

### 4.1 数据库备份
- SQLite: 直接冷备 `simhub.db` 文件。
- MySQL: 使用 `mysqldump`。

### 4.2 存储层备份
- 使用 `mc mirror` 命令同步 MinIO 存储桶内容到备份存储。
