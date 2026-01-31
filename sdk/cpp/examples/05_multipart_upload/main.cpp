#include "simhub/simhub.hpp"
#include <iostream>
#include <fstream>
#include <vector>
#include <chrono>

/**
 * C++ SDK 分片并发上传示例
 */
int main() {
    // 1. 初始化 (仅需一次)
    simhub::Client::GlobalInit();

    // 2. 配置客户端
    simhub::Client client("http://localhost:30030");
    client.setToken("shp_admin_test_token"); // 替换为真实的 Token

    const std::string testFile = "large_test_data.bin";
    
    // 3. 准备测试数据 (15MB)
    std::cout << ">>> 正在准备测试文件 (15MB)..." << std::endl;
    {
        std::ofstream ofs(testFile, std::ios::binary);
        std::vector<char> buffer(1024 * 1024, 'X');
        for(int i=0; i<15; ++i) ofs.write(buffer.data(), buffer.size());
    }

    // 4. 执行并发分片上传
    std::cout << ">>> 开始并发分片上传..." << std::endl;
    auto start = std::chrono::high_resolution_clock::now();

    auto status = client.uploadFileMultipart("scenario", testFile, "C++ 并发压测资源", [](double progress) {
        printf("\r上传进度: %.2f%%", progress * 100);
        fflush(stdout);
    });

    auto end = std::chrono::high_resolution_clock::now();
    std::chrono::duration<double> diff = end - start;

    if (status.ok()) {
        std::cout << "\n>>> 上传成功！" << std::endl;
        std::cout << "耗时: " << diff.count() << " 秒" << std::endl;
        std::cout << "平均速度: " << (15.0 / diff.count()) << " MB/s" << std::endl;
    } else {
        std::cerr << "\n>>> 上传失败: " << status.message << std::endl;
    }

    // 5. 环境清理
    std::remove(testFile.c_str());
    simhub::Client::GlobalCleanup();

    return status.ok() ? 0 : 1;
}
