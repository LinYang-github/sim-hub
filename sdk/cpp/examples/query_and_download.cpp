#include "simhub/client.h"
#include <iostream>
#include <iomanip>

int main() {
    std::cout << "🚀 SimHub C++ SDK Example: Query and Download" << std::endl;

    // 1. Initialize
    simhub::Client::GlobalInit();
    simhub::Client client("http://localhost:30030");

    // 2. Query Resources
    std::cout << "\n--- Querying 'model_glb' Resources ---" << std::endl;
    auto listRes = client.listResources("model_glb");
    
    if (!listRes.ok()) {
        std::cerr << "❌ List failed: " << listRes.message << std::endl;
        simhub::Client::GlobalCleanup();
        return 1;
    }

    auto& resources = listRes.value;
    std::cout << "Found " << resources.size() << " resources:" << std::endl;
    std::cout << std::left << std::setw(38) << "ID" << std::setw(20) << "Name" << "Version" << std::endl;
    std::cout << std::string(70, '-') << std::endl;

    for (const auto& r : resources) {
        std::cout << std::left << std::setw(38) << r.id 
                  << std::setw(20) << r.name 
                  << "v" << r.latest_version.version_num << std::endl;
    }

    if (resources.empty()) {
        std::cout << "No resources found to download." << std::endl;
        simhub::Client::GlobalCleanup();
        return 0;
    }

    // 3. Download the first resource
    const auto& target = resources[0];
    if (target.latest_version.download_url.empty()) {
        std::cout << "\nTarget resource has no download URL." << std::endl;
    } else {
        std::string localPath = "downloaded_" + target.name + ".glb";
        std::cout << "\n--- Downloading: " << target.name << " ---" << std::endl;
        std::cout << "To: " << localPath << std::endl;

        auto status = client.downloadFile(
            target.latest_version.download_url,
            localPath,
            [](double progress) {
                int barWidth = 50;
                std::cout << "[";
                int pos = barWidth * progress;
                for (int i = 0; i < barWidth; ++i) {
                    if (i < pos) std::cout << "=";
                    else if (i == pos) std::cout << ">";
                    else std::cout << " ";
                }
                std::cout << "] " << int(progress * 100.0) << " %\r";
                std::cout.flush();
            }
        );
        std::cout << std::endl;

        if (status.ok()) {
            std::cout << "✅ Download Successful!" << std::endl;
        } else {
            std::cerr << "❌ Download Failed: " << status.message << std::endl;
        }
    }

    // Cleanup
    simhub::Client::GlobalCleanup();
    return 0;
}
