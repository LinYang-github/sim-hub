# SimHub C++ SDK

SimHub 官方 C++ 集成开发包。支持多种上传模式，提供企业级的资源管理集成能力。

## 目录结构
- `include/simhub/`: 公共头文件
- `src/`: SDK 源代码
- `examples/`: 集成示例
  - `01_simple_upload`: 使用预签名 URL 进行基础 HTTP 上传（轻量级，无 AWS 依赖）
  - `02_sts_upload`: 使用 STS 凭证进行 S3 原生上传（高性能，依赖 AWS SDK）

## 构建指南

### 环境要求
- CMake 3.10+
- libcurl
- nlohmann_json (SDK 已内置)
- (可选) AWS SDK for C++ (用于 STS 模式)

### 构建步骤
```bash
mkdir build && cd build
# 基础构建 (仅 HTTP 模式)
cmake .. 
# 启用 AWS SDK 支持 (STS 模式)
cmake .. -DUSE_AWS_SDK=ON 
make
```

## 快速上手

```cpp
#include <simhub/client.h>

int main() {
    // 初始化客户端
    simhub::Client client("http://localhost:30030");

    // 1. 请求上传
    simhub::UploadTokenRequest req;
    req.resource_type = "scenario";
    req.filename = "my_data.zip";
    auto ticket = client.requestUploadToken(req);

    // 2. 上传文件
    client.uploadFileSimple(ticket.presigned_url, "path/to/my_data.zip");

    // 3. 确认完成
    simhub::ConfirmUploadRequest confirm;
    confirm.ticket_id = ticket.ticket_id;
    confirm.name = "MyScenario";
    client.confirmUpload(confirm);
}
```
