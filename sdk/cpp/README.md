# SimHub C++ SDK

[![C++ Standard](https://img.shields.io/badge/C%2B%2B-17%2F20-blue.svg)](https://en.wikipedia.org/wiki/C%2B%2B17)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows%20%7C%20macOS-lightgrey.svg)](#)

SimHub C++ SDK 是一个高性能、低侵入、ABI 稳定的仿真资源管理客户端。它专为大型仿真引擎集成设计，解决了 C++ 库集成中常见的二进制不兼容和复杂的依赖问题。

## 🌟 核心特性

- **Single-Header (STB-style)**: 只需包含一个 `simhub.hpp` 即可完成集成，逻辑解耦。
- **ABI Stability**: 使用 Handle/PImpl 模式，确保跨编译器、跨标准库版本的二进制兼容性。
- **高性能上传**: 支持简单上传 (Simple) 与 分片断点续传 (Multipart)，支持 libcurl 和 AWS SDK (STS) 双模式。
- **零配置初始化**: 自动管理内部组件生命周期。
- **现代 C++ 接口**: 虽然底层 ABI 稳定，但对外提供易用的 `std::string`, `std::vector` 和 `std::map` 封装。

## 🚀 快速开始

### 1. 引入 SDK
将 `sdk/cpp/include/simhub/simhub.hpp` 拷贝到你的项目中。

在项目中的 **一个** 源文件中定义实现宏：
```cpp
#define SIMHUB_IMPLEMENTATION
#include "simhub.hpp"
```

### 2. 基础调用示例
```cpp
#include "simhub.hpp"
#include <iostream>

int main() {
    // 全局初始化 (仅需一次)
    simhub::Client::GlobalInit();

    simhub::Client client("http://localhost:30030");

    // 列出所有模型资源
    auto result = client.listResources("model_glb");
    if (result.ok()) {
        for (const auto& res : result.value) {
            std::cout << "Resource: " << res.name() << " [" << res.id() << "]" << std::endl;
        }
    }

    // 上传文件
    client.uploadFileSimple("documents", "report.pdf", "Mission Report");

    simhub::Client::GlobalCleanup();
    return 0;
}
```

## 🛠️ 构建与依赖

### 外部依赖
- **libcurl** (必须): 用于处理 HTTP/HTTPS 通信。
- **nlohmann/json** (必须): 已内置或需在头文件前引入 `json.hpp`。
- **aws-sdk-cpp-s3** (可选): 如果需要 STS 直传模式，请在编译时定义 `-DUSE_AWS_SDK`。

### CMake 配置
推荐使用 CMake 进行集成。SDK 提供了一个标准的 CMake 目标：

```cmake
# 在你的 CMakeLists.txt 中
include_directories(path/to/simhub/include)
add_executable(my_app main.cpp)
target_link_libraries(my_app PRIVATE CURL::libcurl)
# 如果使用 AWS
# target_compile_definitions(my_app PRIVATE USE_AWS_SDK)
# target_link_libraries(my_app PRIVATE ...AWS_LIBRARIES...)
```

## 📂 示例目录

- `01_quickstart`: 最简单的资源列表查询。
- `02_advanced_upload`: 包含进度回调的分片上传示例。
- `03_download_manager`: 资源下载与本地缓存管理。

---
© 2026 SimHub Team.
