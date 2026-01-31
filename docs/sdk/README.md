# SimHub SDKs 统一门户

欢迎使用 SimHub SDK！我们为多种主流语言提供了原生支持，助您轻松集成 SimHub 的资源管理、仿真调度与数据流转能力。

所有 SDK 均遵循 **统一 API 规范** 与 **[语义化版本控制 (SemVer)](https://semver.org/)**，确保跨语言体验的一致性。

## 📚 语言支持矩阵

| 语言 | 源码路径 | 版本 | 核心特性 | 推荐场景 |
| :--- | :--- | :--- | :--- | :--- |
| **Go** | [`sdk/go`](../../sdk/go) | v1.0.0 | 原生并发、极低开销 | 后端微服务、高性能中间件 |
| **Python** | [`sdk/python`](../../sdk/python) | v1.0.0 | 动态类型、简洁易用 | 脚本自动化、AI/ML 训练流水线 |
| **Java** | [`sdk/java`](../../sdk/java) | v1.0.0 | 强类型、企业级生态 | 大型业务系统、Android 集成 |
| **C++** | [`sdk/cpp`](../../sdk/cpp) | v1.0.0 | ABI 稳定、极致性能 | 仿真引擎集成、嵌入式设备 |

---

## 🚀 核心功能概览

所有官方 SDK 均实现了以下核心 Use Cases：

1.  **资源发现 (Discovery)**
    *   `listResources()`: 支持按类型、分类、关键词等多维检索。
    *   `getResource()`: 获取资源详情及其版本历史。
2.  **数据传输 (Transfer)**
    *   `uploadFileMultipart()`: **并发分片上传**，支持断点续传与大文件（TB级）秒传。
    *   `downloadFile()`: 支持进度回调的流式下载。
3.  **权限管控 (Auth)**
    *   统一支持 Token (Bearer) 鉴权。
    *   自动处理 401/403 状态码与 Token 刷新（部分实现）。

---

## 🛠️ 快速开始 (Quick Start)

### 1. 获取 API Token
在使用 SDK 前，请确保已拥有有效的 Access Token。
*   **Web 控制台**: 登录 SimHub ->以此用户身份 -> 生成 Token。
*   **API**:
    ```bash
    curl -X POST http://<simhub-host>/api/v1/auth/tokens \
      -H "Authorization: Bearer <your-session-token>" \
      -d '{"name": "SDK-Token", "expire_days": 365}'
    ```

### 2. 初始化客户端

#### Go
```go
import "simhub/sdk/go"

client := simhub.NewClient("http://localhost:30030", "your_token")
client.SetConcurrency(8) // 设置最大并发数
```

#### Python
```python
from simhub.client import SimHubClient

client = SimHubClient("http://localhost:30030", "your_token", concurrency=4)
```

#### Java
```java
SimHubClient client = new SimHubClient("http://localhost:30030", "your_token");
```

#### C++
```cpp
#include "simhub/simhub.hpp"

simhub::Client client("http://localhost:30030");
client.setToken("your_token");
```

---

## 📦 示例代码

每个 SDK 目录下均包含完整的 `examples/`，涵盖：
*   **基础 CRUD**: 资源的增删改查。
*   **大文件上传**: 演示如何高效上传 10GB+ 的仿真数据。
*   **数据包下载**: 一键获取资源及其完整依赖树。

请参考各 SDK 目录下的 `README.md` 获取详细编译与运行指南。
