#include "simhub/client.h"
#include <iostream>
#include <vector>
#include <thread>
#include <atomic>
#include <chrono>

using namespace simhub;

void worker(const std::string& baseUrl, int reqCount, std::atomic<int>& success, std::atomic<int>& failure) {
    Client client(baseUrl);
    for (int i = 0; i < reqCount; ++i) {
        UploadTokenRequest req;
        req.resource_type = "scenario";
        req.filename = "stress_client.bin";
        req.mode = "presigned";

        auto res = client.requestUploadToken(req);
        if (res.ok()) {
            success++;
        } else {
            failure++;
        }

        // Simulating some read activity
        auto getRes = client.getResource("any-valid-uuid-or-just-random");
        // We don't care if get fails as long as network doesn't crash
    }
}

int main(int argc, char** argv) {
    Client::GlobalInit();

    std::string baseUrl = "http://localhost:30030";
    int numThreads = 10;
    int reqPerThread = 50;

    std::cout << "Starting C++ SDK Stress Test..." << std::endl;
    std::cout << "Threads: " << numThreads << ", Req/Thread: " << reqPerThread << std::endl;

    std::atomic<int> success{0};
    std::atomic<int> failure{0};
    std::vector<std::thread> threads;

    auto start = std::chrono::high_resolution_clock::now();

    for (int i = 0; i < numThreads; ++i) {
        threads.emplace_back(worker, baseUrl, reqPerThread, std::ref(success), std::ref(failure));
    }

    for (auto& t : threads) {
        t.join();
    }

    auto end = std::chrono::high_resolution_clock::now();
    std::chrono::duration<double> diff = end - start;

    std::cout << "\n--- SDK Stress Test Result ---" << std::endl;
    std::cout << "Total Requests: " << (success + failure) << std::endl;
    std::cout << "Success:        " << success << std::endl;
    std::cout << "Failure:        " << failure << std::endl;
    std::cout << "Time Taken:     " << diff.count() << " s" << std::endl;
    std::cout << "RPS:            " << (success + failure) / diff.count() << std::endl;

    Client::GlobalCleanup();
    return 0;
}
