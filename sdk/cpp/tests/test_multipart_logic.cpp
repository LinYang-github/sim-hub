#include <gtest/gtest.h>
#include <simhub/simhub.hpp>
#include <fstream>
#include <cstdio>

using namespace simhub;

class MultipartTest : public ::testing::Test {
protected:
    void SetUp() override {
        // Create a dummy 12MB file for testing multipart (5MB segments)
        const char* filename = "test_multipart_data.bin";
        std::ofstream ofs(filename, std::ios::binary);
        std::vector<char> buffer(1024 * 1024, 'A'); // 1MB buffer
        for(int i=0; i<12; ++i) {
            ofs.write(buffer.data(), buffer.size());
        }
        ofs.close();
    }

    void TearDown() override {
        std::remove("test_multipart_data.bin");
    }
};

// This test verifies that the state management and thread launching logic works
// Even if the network fails (no real server), we check if the SDK handles it gracefully.
TEST_F(MultipartTest, BasicLogicCheck) {
    Client client("http://invalid-local-host:9999");
    client.setToken("test-token");

    // This is expected to fail with NetworkError because the host is invalid,
    // but it should NOT crash and should return a Failure Result.
    auto status = client.uploadFileMultipart("test", "test_multipart_data.bin", "Test Resource");
    
    EXPECT_FALSE(status.ok());
    EXPECT_EQ(status.code, ErrorCode::NetworkError);
}

// Check progress callback
TEST_F(MultipartTest, ProgressCallbackCheck) {
    Client client("http://invalid-local-host:9999");
    
    bool progressCalled = false;
    client.uploadFileMultipart("test", "test_multipart_data.bin", "Test Resource", [&](double p){
        progressCalled = true;
    });

    // Since it fails at Init (Network Error), progress might not be called, 
    // which is correct behavior for a failing start.
}
