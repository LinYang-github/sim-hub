#include <iostream>
#define SIMHUB_IMPLEMENTATION
#include "simhub/simhub.hpp"
#include <filesystem>

/**
 * 示例 03: 资源包递归下载
 * 展示如何通过单个 ID 自动解析依赖树并一键下载所有关联资产。
 * 对大型仿真场景一键部署非常有用。
 */

namespace fs = std::filesystem;

int main(int argc, char* argv[]) {
    simhub::Client::GlobalInit();
    simhub::Client client("http://localhost:30030");

    if (argc < 2) {
        std::cout << "Usage: " << argv[0] << " <resource_id>" << std::endl;
        // 尝试获取一个存在的 ID 作为演示
        auto list = client.listResources();
        if (list.ok() && !list.value.empty()) {
            std::cout << "Suggested ID: " << list.value[0].id << std::endl;
        }
        return 1;
    }

    std::string resId = argv[1];
    std::string downloadDir = "downloads";
    
    // 1. 确保目录存在
    if (!fs::exists(downloadDir)) {
        fs::create_directory(downloadDir);
    }

    std::cout << "Resolving and downloading bundle for Resource: " << resId << std::endl;
    std::cout << "Target Directory: " << fs::absolute(downloadDir) << std::endl;

    // 2. 一键下载
    // 该方法内部会自动递归访问 getResourceDependencies -> downloadFile
    auto status = client.downloadBundle(resId, downloadDir);

    if (status.ok()) {
        std::cout << "✅ Bundle download successful!" << std::endl;
        std::cout << "Contents of " << downloadDir << ":" << std::endl;
        for (const auto& entry : fs::directory_iterator(downloadDir)) {
            std::cout << "  - " << entry.path().filename() << std::endl;
        }
    } else {
        std::cerr << "❌ Bundle download failed: " << status.message << std::endl;
    }

    simhub::Client::GlobalCleanup();
    return 0;
}
