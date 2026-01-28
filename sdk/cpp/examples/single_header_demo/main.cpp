#include <iostream>
#include <fstream>
#include "simhub/simhub.hpp"
#include <iomanip>

// Simple progress bar
void printProgress(double progress) {
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

int main(int argc, char* argv[]) {
    // 1. Global Init
    simhub::Client::GlobalInit();

    std::cout << "SimHub SDK Demo" << std::endl;
    std::cout << "----------------" << std::endl;

    // 2. Create Client
    std::string baseUrl = "http://localhost:30030";
    if (argc > 1) baseUrl = argv[1];
    
    simhub::Client client(baseUrl);

    // 3. List Resources
    std::cout << "\n[1] Listing resources..." << std::endl;
    auto listRes = client.listResources();
    if (!listRes.ok()) {
        std::cerr << "Failed to list resources: " << listRes.message << std::endl;
        return 1;
    }

    std::cout << "Found " << listRes.value.size() << " resources:" << std::endl;
    for (const auto& res : listRes.value) {
        std::cout << " - " << res.name << " (" << res.latest_version.file_size << " bytes)" << std::endl;
    }

    // 4. Create a dummy file for upload
    std::string dummyFile = "test_upload.txt";
    {
        std::ofstream f(dummyFile);
        f << "This is a test file uploaded via SimHub C++ SDK Single Header." << std::endl;
        for(int i=0; i<100; i++) f << "Line " << i << " data..." << std::endl;
    }

    // 5. Upload File
    std::cout << "\n[2] Uploading " << dummyFile << "..." << std::endl;
    auto uploadStatus = client.uploadFileSimple("documents", dummyFile, "SDK Upload Test", printProgress);
    
    if (uploadStatus.ok()) {
        std::cout << "\nUpload successful!" << std::endl;
    } else {
        std::cerr << "\nUpload failed: " << uploadStatus.message << std::endl;
    }

    // 6. Cleanup
    simhub::Client::GlobalCleanup();
    return 0;
}
