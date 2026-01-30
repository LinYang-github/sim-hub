#include <iostream>
#define SIMHUB_IMPLEMENTATION
#include "simhub/simhub.hpp"
#include <vector>
#include <future>

/**
 * 示例 02: 异步操作
 * 展示如何利用 std::future 实现非阻塞的 API 调用，提升集成响应速度。
 */

int main() {
    simhub::Client::GlobalInit();
    simhub::Client client("http://localhost:30030");

    std::cout << "Starting multiple async requests..." << std::endl;

    // 1. 同时发起多个异步请求
    auto f1 = client.listResourcesAsync("model_glb");
    auto f2 = client.listResourcesAsync("scenario_json");
    auto f3 = client.listCategoriesAsync("model_glb");

    std::cout << "Doing other work while waiting for network..." << std::endl;
    std::this_thread::sleep_for(std::chrono::milliseconds(500)); 

    // 2. 等待结果并处理
    auto res1 = f1.get();
    auto res2 = f2.get();
    
    if (res1.ok()) {
        std::cout << "Async Models found: " << res1.value.size() << std::endl;
    }

    if (res2.ok()) {
        std::cout << "Async Scenarios found: " << res2.value.size() << std::endl;
    }

    // 3. 异步下载演示
    if (res1.ok() && !res1.value.empty()) {
        auto& res = res1.value[0];
        if (!res.latest_version.download_url.empty()) {
            std::cout << "Starting async download for: " << res.name << std::endl;
            auto fDl = client.downloadFileAsync(res.latest_version.download_url, "async_download.zip");
            
            // 可以在此处检查下载是否完成
            if (fDl.wait_for(std::chrono::seconds(0)) == std::future_status::ready) {
                std::cout << "Download finished instantly!" << std::endl;
            } else {
                std::cout << "Download in progress, waiting..." << std::endl;
                auto status = fDl.get();
                if (status.ok()) std::cout << "Download complete!" << std::endl;
            }
        }
    }

    simhub::Client::GlobalCleanup();
    return 0;
}
