#include "simhub/client.h"
#include <iostream>
#include <fstream>
#include <thread>

// Utility to create a dummy large file
void createDummyFile(const std::string& path, size_t sizeMB) {
    std::ofstream ofs(path, std::ios::binary);
    std::vector<char> buffer(1024 * 1024, 'A'); // 1MB buffer
    for (size_t i = 0; i < sizeMB; ++i) {
        ofs.write(buffer.data(), buffer.size());
    }
}

int main() {
    std::cout << "🚀 SimHub C++ SDK Example: Large File Upload" << std::endl;

    // 1. Initialize Global SDK Resources
    simhub::Client::GlobalInit();

    // 2. Create Client (pointing to API Node)
    simhub::Client client("http://localhost:30030");

    // 3. Prepare a test file (e.g., 15MB to trigger multipart)
    std::string filePath = "large_model.glb";
    std::cout << "Creating dummy file: " << filePath << " ..." << std::endl;
    createDummyFile(filePath, 15); 

    // 4. Execute Multipart Upload
    std::cout << "Starting multipart upload..." << std::endl;
    auto status = client.uploadFileMultipart(
        "model_glb",          // TypeKey (must match config-api.yaml)
        filePath,             // Local path
        "My 3D Model",        // Resource Name
        [](double progress) { // Progress Callback
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
        std::cout << "✅ Upload Successful!" << std::endl;
    } else {
        std::cerr << "❌ Upload Failed: " << status.message << std::endl;
    }

    // Cleanup
    simhub::Client::GlobalCleanup();
    std::remove(filePath.c_str());
    return 0;
}
