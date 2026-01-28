#include <iostream>
#include <fstream>
#define SIMHUB_IMPLEMENTATION
#include "simhub/simhub.hpp"
#include <iomanip>

/**
 * 示例 02: 进阶上传
 * 展示如何实现带进度条的大文件分片上传。
 */

void printProgressBar(double progress) {
    int barWidth = 40;
    std::cout << "\rUpload Progress: [";
    int pos = barWidth * progress;
    for (int i = 0; i < barWidth; ++i) {
        if (i < pos) std::cout << "=";
        else if (i == pos) std::cout << ">";
        else std::cout << " ";
    }
    std::cout << "] " << std::fixed << std::setprecision(1) << (progress * 100.0) << "% " << std::flush;
}

int main() {
    simhub::Client::GlobalInit();
    simhub::Client client("http://localhost:30030");

    // 1. 准备一个超过 5MB 的测试文件（模拟大文件）
    std::string largeFile = "large_resource.dat";
    std::cout << "Creating dummy large file..." << std::endl;
    {
        std::ofstream f(largeFile, std::ios::binary);
        std::vector<char> dummyData(10 * 1024 * 1024, 'X'); // 10MB
        f.write(dummyData.data(), dummyData.size());
    }

    // 2. 使用分片上传 (uploadFileMultipart)
    // 注意：该方法内部会自动进行：
    // 初始化分片 -> 获取批量预签名URL -> 逐个上传(带重试) -> 合并分片
    std::cout << "Starting multipart upload for " << largeFile << std::endl;
    
    auto status = client.uploadFileMultipart(
        "model_glb",       // 资源类型
        largeFile,         // 本地路径
        "Large Test Model",// 资源名称
        printProgressBar,  // 进度回调
        3                  // 最大重试次数
    );

    std::cout << std::endl; // 结束进度条行

    if (status.ok()) {
        std::cout << "🎉 Upload completed successfully!" << std::endl;
    } else {
        std::cerr << "❌ Upload failed: " << status.message << std::endl;
    }

    // 清理
    std::remove(largeFile.c_str());
    simhub::Client::GlobalCleanup();
    return 0;
}
