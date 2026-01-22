#include <iostream>
#include <fstream>
#include <vector>
#include <sys/stat.h>
#include "simhub_client.h"

// AWS SDK Headers
#include <aws/core/Aws.h>
#include <aws/core/auth/AWSCredentials.h>
#include <aws/s3/S3Client.h>
#include <aws/s3/model/PutObjectRequest.h>
#include <aws/core/utils/memory/stl/AWSStringStream.h>

int main(int argc, char* argv[]) {
    // 1. Prepare Test File
    std::string zipPath = "test.zip";
    if (argc >= 2) zipPath = argv[1];
    
    // Ensure file exists
    struct stat buffer;
    if (stat(zipPath.c_str(), &buffer) != 0) {
        std::ofstream outfile(zipPath);
        outfile << "Real STS Upload Content via AWS SDK" << std::endl;
        outfile.close();
        std::cout << "Created dummy " << zipPath << std::endl;
    }

    SimHubClient client("http://localhost:30030");
    std::string name = "Real_AWS_SDK_Test";
    
    // 2. Init AWS SDK Global
    Aws::SDKOptions options;
    Aws::InitAPI(options);
    { // Scope for AWS objects

        // 3. Request Credentials from SimHub
        std::cout << "[Step 1] Requesting STS Credentials..." << std::endl;
        json reqToken = {
            {"resource_type", "scenario"},
            {"filename", name + ".zip"},
            {"size", 1024},
            {"checksum", "none"},
            {"mode", "sts"} 
        };

        json res = client.post("/api/v1/integration/upload/token", reqToken);

        if (res.contains("error")) {
            std::cerr << "Error: " << res["error"] << std::endl;
            return 1;
        }

        json creds = res["credentials"];
        std::string ak = creds["access_key"];
        std::string sk = creds["secret_key"];
        std::string token = creds["session_token"];
        std::string bucket = res["bucket"];
        std::string objectKey = res["object_key"];
        std::string ticketId = res["ticket_id"];

        std::cout << "[Step 2] Got Credentials for " << bucket << "/" << objectKey << std::endl;

        // 4. Configure S3 Client with Temporary Credentials
        Aws::Auth::AWSCredentials awsCreds(ak.c_str(), sk.c_str(), token.c_str());
        Aws::Client::ClientConfiguration clientConfig;
        clientConfig.endpointOverride = "localhost:9000"; // Point to local MinIO
        clientConfig.scheme = Aws::Http::Scheme::HTTP;
        clientConfig.verifySSL = false; // For local dev
        
        // Fix for MinIO requiring path-style access usually
        // But AWS SDK 1.9+ defaults to virtual-hosted. 
        // We might need forcePathStyle for older SDKs or specific configs. 
        // Using standard S3Client for now.

        Aws::S3::S3Client s3_client(awsCreds, clientConfig, 
            Aws::Client::AWSAuthV4Signer::PayloadSigningPolicy::Never, // MinIO sometimes happier without payload signing on http
            false); // useVirtualAddressing = false (Path Style)

        // 5. Upload File
        std::cout << "[Step 3] Uploading file via AWS SDK..." << std::endl;
        Aws::S3::Model::PutObjectRequest request;
        request.SetBucket(bucket.c_str());
        request.SetKey(objectKey.c_str());

        std::shared_ptr<Aws::IOStream> inputData = Aws::MakeShared<Aws::FStream>("SampleAllocationTag",
            zipPath.c_str(), 
            std::ios_base::in | std::ios_base::binary);

        request.SetBody(inputData);

        auto outcome = s3_client.PutObject(request);

        if (!outcome.IsSuccess()) {
             std::cerr << "S3 Upload Error: " << outcome.GetError().GetMessage() << std::endl;
             // Don't return 1 yet, try to clean up? No, fail.
             // return 1; 
             // We continue to see if Confirm fails (it will validly fail if upload failed really)
        } else {
             std::cout << "S3 Upload Successful!" << std::endl;
        }

        // 6. Confirm to SimHub
        if (outcome.IsSuccess()) {
            std::cout << "\n[Step 4] Confirming upload..." << std::endl;
            json reqConfirm = {
                {"ticket_id", ticketId},
                {"type_key", "scenario"},
                {"name", name},
                {"owner_id", "aws-sdk-cpp-client"},
                {"size", 1024}, 
                {"extra_meta", {{"method", "aws_sdk_cpp"}}}
            };
            
            json resConfirm = client.post("/api/v1/integration/upload/confirm", reqConfirm);
            if (resConfirm.contains("error")) {
                std::cerr << "Confirmation failed: " << resConfirm["error"] << std::endl;
            } else {
                std::cout << "Success! Scenario registered via REAL AWS SDK." << std::endl;
            }
        }

    } // End AWS Scope
    Aws::ShutdownAPI(options);
    
    return 0;
}
