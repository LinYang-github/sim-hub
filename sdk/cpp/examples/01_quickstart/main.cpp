#include <iostream>
#define SIMHUB_IMPLEMENTATION
#include "simhub/simhub.hpp"
#include <iomanip>

/**
 * 示例 01: 快速起步
 * 展示如何连接服务器、列举资源并读取其元数据。
 */

void printResourceInfo(const simhub::Resource& res) {
    std::cout << "[Resource Info]" << std::endl;
    std::cout << " - Name:     " << res.name << std::endl;
    std::cout << " - ID:       " << res.id << std::endl;
    std::cout << " - Type:     " << res.type_key << std::endl;
    std::cout << " - Category: " << res.category_id << std::endl;
    std::cout << " - Tags:     ";
    for (const auto& tag : res.tags) std::cout << tag << " ";
    std::cout << std::endl;

    if (!res.latest_version.semver.empty()) {
        std::cout << " - Latest:   " << res.latest_version.semver << " (" << res.latest_version.state << ")" << std::endl;
        std::cout << " - Size:     " << res.latest_version.file_size << " bytes" << std::endl;
    }
    std::cout << "-----------------------------------" << std::endl;
}

int main(int argc, char* argv[]) {
    // 全局初始化
    simhub::Client::GlobalInit();

    std::string baseUrl = "http://localhost:30030";
    if (argc > 1) baseUrl = argv[1];
    
    std::cout << "Connecting to " << baseUrl << "..." << std::endl;
    simhub::Client client(baseUrl);

    // 1. 列举资源
    auto result = client.listResources();
    if (result.ok()) {
        std::cout << "Found " << result.value.size() << " resources.\n" << std::endl;
        for (const auto& res : result.value) {
            printResourceInfo(res);
        }
    } else {
        std::cerr << "List failed: " << result.message << std::endl;
    }

    simhub::Client::GlobalCleanup();
    return 0;
}
