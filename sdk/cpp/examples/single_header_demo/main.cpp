#include <iostream>
#include <fstream>
#define SIMHUB_IMPLEMENTATION
#include "simhub/simhub.hpp"
#include <iomanip>

/**
 * 示例 01: 快速起步
 * 展示如何连接服务器、列举资源并读取其元数据。
 */

void printResourceInfo(const simhub::Resource& res) {
    if (!res.isValid()) return;
    
    std::cout << "[Resource Info]" << std::endl;
    std::cout << " - Name:  " << res.name() << std::endl;
    std::cout << " - ID:    " << res.id() << std::endl;
    std::cout << " - Owner: " << res.ownerId() << std::endl;
    std::cout << " - Tags:  ";
    for (const auto& tag : res.tags()) std::cout << tag << " ";
    std::cout << std::endl;

    auto ver = res.latestVersion();
    if (ver.isValid()) {
        std::cout << " - Latest Version: " << ver.semver() << " (" << ver.state() << ")" << std::endl;
        std::cout << " - File Size:      " << ver.fileSize() << " bytes" << std::endl;
    }
    std::cout << "-----------------------------------" << std::endl;
}

int main(int argc, char* argv[]) {
    // 全局初始化，确保底层资源准备就绪
    simhub::Client::GlobalInit();

    std::string baseUrl = "http://localhost:30030";
    if (argc > 1) baseUrl = argv[1];
    
    std::cout << "Connecting to " << baseUrl << "..." << std::endl;
    simhub::Client client(baseUrl);

    // 1. 列举所有资源
    auto listRes = client.listResources();
    if (listRes.ok()) {
        std::cout << "Found " << listRes.value.size() << " resources.\n" << std::endl;
        for (const auto& res : listRes.value) {
            printResourceInfo(res);
        }
    } else {
        std::cerr << "List failed: " << listRes.message << std::endl;
    }

    // 2. 获取单个资源
    if (!listRes.value.empty()) {
        std::string firstId = listRes.value[0].id();
        std::cout << "\nFetching resource by ID: " << firstId << std::endl;
        auto singleRes = client.getResource(firstId);
        if (singleRes.ok()) {
            std::cout << "Get successful. Name: " << singleRes.value.name() << std::endl;
        }
    }

    simhub::Client::GlobalCleanup();
    return 0;
}
