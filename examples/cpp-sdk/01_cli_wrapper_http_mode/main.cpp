#include <iostream>
#include <sys/stat.h>
#include "simhub_client.h"

int main(int argc, char* argv[]) {
    if (argc < 4) {
        std::cout << "Usage: ./simhub_cli <api_url> <scenario_name> <zip_path>" << std::endl;
        std::cout << "Example: ./simhub_cli http://localhost:30030 MyTestScenario ./test.zip" << std::endl;
        return 1;
    }

    std::string apiUrl = argv[1];
    std::string name = argv[2];
    std::string zipPath = argv[3];

    // Check if zip exists
    struct stat buffer;
    if (stat(zipPath.c_str(), &buffer) != 0) {
        std::cerr << "Error: File " << zipPath << " does not exist." << std::endl;
        return 1;
    }

    SimHubClient client(apiUrl);
    
    try {
        client.uploadScenario(name, zipPath);
    } catch (const std::exception& e) {
        std::cerr << "Exception: " << e.what() << std::endl;
        return 1;
    }

    return 0;
}
